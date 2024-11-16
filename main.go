package main

import (
	"encoding/csv"
	"fmt"
	"image"
	"io"
	"log"
	"net/http"
	"os"
	"runtime"
	"sync"
	"time"

	"github.com/corona10/goimagehash"
	"github.com/xuri/excelize/v2"
	_ "image/jpeg"
	_ "image/png"
)

var httpClient = &http.Client{
	Timeout: 180 * time.Second, // Set HTTP timeout
}

type ImageRecord struct {
	ID           string
	StoreID      string
	FrontlinerID string
	PhotoURL     string
}

type DuplicateRecord struct {
	FrontlinerID       string
	DuplicateImageURLs []string
	DuplicateIDs       []string
}

type FailedRecord struct {
	ID           string
	StoreID      string
	FrontlinerID string
	PhotoURL     string
}

// downloadImage fetches and decodes an image from a URL with retry logic.
func downloadImage(url string) (image.Image, error) {
	var img image.Image
	var _ error

	maxRetries := 3
	for attempts := 1; attempts <= maxRetries; attempts++ {
		log.Print("Downloading image....")
		resp, err := httpClient.Get(url)
		if err != nil {
			log.Printf("Attempt %d: Failed to download image from %s: %v\n", attempts, url, err)
			time.Sleep(120 * time.Second) // Wait before retrying
			continue
		}
		defer func(Body io.ReadCloser) {
			err := Body.Close()
			if err != nil {

			}
		}(resp.Body)

		img, _, err = image.Decode(resp.Body)
		if err == nil {
			return img, nil
		}

		log.Printf("Attempt %d: Failed to decode image from %s: %v\n", attempts, url, err)
		time.Sleep(120 * time.Second) // Wait before retrying
	}

	return nil, fmt.Errorf("failed to download image from %s after %d attempts", url, maxRetries)
}

// parseCSV reads image records from a CSV file.
func parseCSV(filename string) ([]ImageRecord, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer func(file *os.File) {
		err := file.Close()
		if err != nil {

		}
	}(file)

	reader := csv.NewReader(file)
	records, err := reader.ReadAll()
	if err != nil {
		return nil, err
	}

	var imageRecords []ImageRecord
	for i, record := range records {
		if i == 0 {
			continue // Skip header row
		}
		imageRecords = append(imageRecords, ImageRecord{
			ID:           record[0],
			StoreID:      record[1],
			FrontlinerID: record[2],
			PhotoURL:     record[3],
		})
	}

	return imageRecords, nil
}

// detectDuplicates identifies duplicate images and tracks failed records.
func detectDuplicates(imageRecords []ImageRecord, threshold int) ([]DuplicateRecord, []FailedRecord) {
	var hashStore sync.Map
	duplicateMap := make(map[string]map[string]*DuplicateRecord)
	var failedRecords []FailedRecord
	var mu sync.Mutex // Protects shared data (failedRecords)

	// Worker pool
	numWorkers := 10
	jobs := make(chan ImageRecord, len(imageRecords))
	results := make(chan struct{}, len(imageRecords))

	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	log.Printf("Memory Alloc = %v MiB", m.Alloc/1024/1024)

	// Worker function
	worker := func() {
		for record := range jobs {
			img, err := downloadImage(record.PhotoURL)
			if err != nil {
				log.Printf("Failed to download image from %s: %v\n", record.PhotoURL, err)
				mu.Lock()
				failedRecords = append(failedRecords, FailedRecord{
					ID:           record.ID,
					StoreID:      record.StoreID,
					FrontlinerID: record.FrontlinerID,
					PhotoURL:     record.PhotoURL,
				})
				mu.Unlock()
				results <- struct{}{}
				continue
			}

			hash, err := goimagehash.PerceptionHash(img)
			if err != nil {
				log.Printf("Failed to hash image from %s: %v\n", record.PhotoURL, err)
				results <- struct{}{}
				continue
			}

			isDuplicate := false
			hashStore.Range(func(key, value interface{}) bool {
				existingRecord := key.(ImageRecord)
				existingHash := value.(*goimagehash.ImageHash)
				distance, _ := hash.Distance(existingHash)
				if distance < threshold {
					isDuplicate = true

					mu.Lock()
					if _, exists := duplicateMap[record.FrontlinerID]; !exists {
						duplicateMap[record.FrontlinerID] = make(map[string]*DuplicateRecord)
					}
					if _, exists := duplicateMap[record.FrontlinerID][existingRecord.FrontlinerID]; !exists {
						duplicateMap[record.FrontlinerID][existingRecord.FrontlinerID] = &DuplicateRecord{
							FrontlinerID:       record.FrontlinerID,
							DuplicateImageURLs: []string{},
							DuplicateIDs:       []string{},
						}
					}

					duplicateRecord := duplicateMap[record.FrontlinerID][existingRecord.FrontlinerID]
					duplicateRecord.DuplicateImageURLs = append(duplicateRecord.DuplicateImageURLs, record.PhotoURL, existingRecord.PhotoURL)
					duplicateRecord.DuplicateIDs = append(duplicateRecord.DuplicateIDs, record.ID, existingRecord.ID)
					mu.Unlock()
					return false
				}
				return true
			})

			if !isDuplicate {
				hashStore.Store(record, hash)
			}
			results <- struct{}{}
		}
	}

	// Start workers
	for i := 0; i < numWorkers; i++ {
		go worker()
	}

	// Send jobs
	for _, record := range imageRecords {
		jobs <- record
	}
	close(jobs)

	// Wait for results
	for range imageRecords {
		<-results
	}

	var duplicates []DuplicateRecord
	for _, frontlinerRecords := range duplicateMap {
		for _, record := range frontlinerRecords {
			duplicates = append(duplicates, *record)
		}
	}

	return duplicates, failedRecords
}

// writeResultsToExcel writes the duplicate detection results to an Excel file.
func writeResultsToExcel(duplicates []DuplicateRecord, filename string) error {
	f := excelize.NewFile()
	sheet := "Sheet1"

	headers := []string{"frontliner_id", "duplicate image URLs", "id"}
	for col, header := range headers {
		cell, _ := excelize.CoordinatesToCellName(col+1, 1)
		err := f.SetCellValue(sheet, cell, header)
		if err != nil {
			return err
		}
	}

	for i, record := range duplicates {
		err := f.SetCellValue(sheet, fmt.Sprintf("A%d", i+2), record.FrontlinerID)
		if err != nil {
			return err
		}
		err = f.SetCellValue(sheet, fmt.Sprintf("B%d", i+2), fmt.Sprintf("%v", record.DuplicateImageURLs))
		if err != nil {
			return err
		}
		err = f.SetCellValue(sheet, fmt.Sprintf("C%d", i+2), fmt.Sprintf("%v", record.DuplicateIDs))
		if err != nil {
			return err
		}
	}

	return f.SaveAs(filename)
}

// writeFailedRecordsToExcel writes failed download records to an Excel file.
func writeFailedRecordsToExcel(failedRecords []FailedRecord, filename string) error {
	f := excelize.NewFile()
	sheet := "Sheet1"

	headers := []string{"id", "store_id", "frontliner_id", "photo_url"}
	for col, header := range headers {
		cell, _ := excelize.CoordinatesToCellName(col+1, 1)
		err := f.SetCellValue(sheet, cell, header)
		if err != nil {
			return err
		}
	}

	for i, record := range failedRecords {
		err := f.SetCellValue(sheet, fmt.Sprintf("A%d", i+2), record.ID)
		if err != nil {
			return err
		}
		err = f.SetCellValue(sheet, fmt.Sprintf("B%d", i+2), record.StoreID)
		if err != nil {
			return err
		}
		err = f.SetCellValue(sheet, fmt.Sprintf("C%d", i+2), record.FrontlinerID)
		if err != nil {
			return err
		}
		err = f.SetCellValue(sheet, fmt.Sprintf("D%d", i+2), record.PhotoURL)
		if err != nil {
			return err
		}
	}

	return f.SaveAs(filename)
}

func main() {
	csvFilename := "your-file.csv" // change based on the desired file
	duplicatesExcelFilename := "duplicates.xlsx"
	failedExcelFilename := "failed_downloads.xlsx"
	threshold := 1

	start := time.Now()

	imageRecords, err := parseCSV(csvFilename)
	if err != nil {
		log.Fatalf("Failed to read CSV file: %v\n", err)
	}

	duplicates, failedRecords := detectDuplicates(imageRecords, threshold)
	fmt.Printf("Duplicate detection complete.\n")

	err = writeResultsToExcel(duplicates, duplicatesExcelFilename)
	if err != nil {
		log.Fatalf("Failed to write Excel file for duplicates: %v\n", err)
	}
	fmt.Printf("Duplicates written to %s\n", duplicatesExcelFilename)

	err = writeFailedRecordsToExcel(failedRecords, failedExcelFilename)
	if err != nil {
		log.Fatalf("Failed to write Excel file for failed downloads: %v\n", err)
	}
	fmt.Printf("Failed downloads written to %s\n", failedExcelFilename)

	fmt.Printf("Time taken: %v\n", time.Since(start))
}
