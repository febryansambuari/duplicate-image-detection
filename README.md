# Duplicate Image Detection in Go

## Overview
This project is a **duplicate image detection tool** built using **Golang**. It reads image metadata from a CSV file, downloads images from URLs, and uses **perceptual hashing** to identify duplicates. The results, including duplicates and failed downloads, are exported as Excel files for analysis.

---

## Features
1. **CSV Parsing**:
   - Reads metadata such as `id`, `store_id`, `frontliner_id`, and `photo_url` from a CSV file.
   > id, store_id, frontliner_id, and photo_url is optional, and you can adjust based on you need.

2. **Image Downloading with Retry Mechanism**:
   - Handles intermittent network issues with retry and timeout capabilities.

3. **Duplicate Image Detection**:
   - Compares images using **perceptual hashing** to detect visual similarities.
   - Configurable threshold for sensitivity.

4. **Failed Downloads Logging**:
   - Tracks and logs images that fail to download due to invalid URLs or network issues.

5. **Excel Export**:
   - Outputs results in two Excel files:
     - **Duplicates**: Contains detected duplicate image information.
     - **Failed Downloads**: Lists images that could not be downloaded.

6. **Concurrency for Performance**:
   - Processes images in parallel using Goroutines for scalability and speed.

---

## Why This Project?

Duplicate detection is essential in many industries, such as:
- **Retail and Marketing**: Ensuring only unique product photos are uploaded.
- **Media Management**: Cleaning up archives by identifying duplicates.

This project demonstrates:
- Efficient use of Go's concurrency model.
- Practical error handling for network-intensive applications.
- Integration with libraries for image processing and file I/O.

---

## Installation

1. **Prerequisites**:
   - Make sure you have Go installed in your computer (v1.20+ recommended).

2. **Clone the Repository**:
   ```bash
   git clone https://github.com/febryansambuari/duplicate-image-detection.git
   cd duplicate-image-detection
   
3. **Install Dependencies:**
   ```bash
   go mod tidy

4. **Run the project:**
    ```bash
    go run main.go

## Usage

1. **Prepare a CSV file with the following headers**:
    ```bash
    id,store_id,frontliner_id,photo_url
    ```

    - Example:
        ```bash
        1,123,456,https://example.com/image1.jpg
        2,123,789,https://example.com/image2.jpg
        ```

2. **Place the CSV file in the project directory**
3. **Update the main.go file to point to your CSV filename**:
    ```bash
    csvFilename := "your-file.csv" 
    ```
4. **Run the program**:
    ```bash
    go run main.go
    ```

## Project Structure

```bash
├── main.go                # Main application file
├── go.mod                 # Dependency management
├── duplicates.xlsx        # Output: Duplicate detection results
├── failed_downloads.xlsx  # Output: Failed downloads
└── your-file.csv          # Input: Image metadata
```

## Learning Goals

This project was built to:

- **Understand Go concurrency**: Use Goroutines and WaitGroups for parallel processing.
- **Implement error handling**: Gracefully handle network and decoding issues.
- **Work with file I/O**: Read/write CSV and Excel files in Go.
- **Apply image hashing**: Learn how perceptual hashing can identify visually similar images

## Future Enhancements

1. **Support for Multiple Image Formats:**
    - Add support for more formats like GIF, BMP, etc.
2. **Web Interface**:
    - Develop a simple web UI to upload files and view results.
3. **Dockerization**:
    - Create a Docker image for easier deployment.
4. **Database Integration**:
    - Store metadata and results in a database for large-scale usage.