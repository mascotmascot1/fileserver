[![Go](https://img.shields.io/badge/Go-1.20%2B-007acc?style=for-the-badge)](https://go.dev)
[![Release](https://img.shields.io/github/release/mascotmascot1/fileserver.svg?label=Release&color=007acc&style=for-the-badge)](https://github.com/mascotmascot1/fileserver/releases/latest)
[![License: MIT](https://img.shields.io/badge/License-MIT-007acc?style=for-the-badge)](https://opensource.org/licenses/MIT)

# 📁 Go Fileserver

A simple, secure, and configurable file server written in Go. Designed for easy file sharing within a local network, but can also be safely exposed to the internet.

## ✨ Features

  * **📤 Upload multiple files** at once via multipart/form-data.
  * **📥 Download individual files** by name.
  * **📋 List all available files** in the storage directory.
  * **🔒 Secure by design**, with built-in protection against Path Traversal attacks.
  * **⚙️ Fully configurable** via a single `fileserver.yaml` file.
  * **⚡ Efficient and lightweight**, with minimal resource usage and robust error handling.

-----

## 🚀 Getting Started

### 1\. Configuration

Before running the server, create a `fileserver.yaml` file in the root of the project. You can start with the example below.
If this file is not found, the server will start with default settings defined in internal/config/config.go. 

```yaml
server:
  # The network address for the server (format: "host:port").
  # An empty host (e.g., ":8090") means listening on all available network interfaces (0.0.0.0).
  address: ":8090"
  
  # Connection timeouts to protect against slow clients and resource exhaustion.
  # Valid time units are "ns", "ms", "s", "m", "h" (e.g., "500ms", "1m30s").
  readTimeout: 5s
  writeTimeout: 10s
  idleTimeout: 30s

uploader:
  # The directory where uploaded files will be stored.
  storageDir: "storage"

  # The maximum permitted size of a single upload request, in megabytes (MB).
  maxUploadSizeMB: 3072 
  
  # The maximum amount of memory (in MB) to use for parsing a multipart form
  # before spooling file parts to temporary files on disk.
  maxFormMemSizeMB: 32
```

---
## 🪵 Logging

The server writes log entries to two destinations simultaneously:

* **Standard Output (stdout):** For real-time monitoring in your console.
* **`server.log` file:** A persistent log file that is created in the same directory where the executable is run. This file is appended to on subsequent runs.
---

### 2\. Run the Server

Navigate to the project's root directory and run the application:

```bash
go run ./cmd/fileserver/
```

The server will start on the address specified in your `fileserver.yaml`.

-----

## 🛠️ API Usage

You can interact with the server using any HTTP client, such as `curl` or Postman.

### Upload a File

To upload a file, send a `POST` request with `multipart/form-data` to the `/upload` endpoint.

```bash
# Replace 'path/to/your/file.txt' with the actual file path.
curl -X POST -F "myFile=@/path/to/your/file.txt" http://localhost:8090/upload
```

### Download a File

To download a file, send a `GET` request to the `/download/` endpoint followed by the filename.

```bash
# This will save the file as 'downloaded-file.zip' in your current directory.
curl -o downloaded-file.zip http://localhost:8090/download/file.zip
```

### List All Files

To get a list of all available files, send a `GET` request to `/download/list.txt`.

```bash
curl http://localhost:8090/download/list.txt
```

-----

## 📦 Building for Production

To create a standalone executable, run the following command from the project root:

```bash
# For Linux/macOS
go build -o fileserver ./cmd/fileserver/

# For Windows
go build -o fileserver.exe ./cmd/fileserver/
```

This will create a `fileserver` (or `fileserver.exe`) binary that you can run anywhere.

-----

## 📜 Licence

This project is licensed under the MIT Licence. See the `LICENSE` file for details.
