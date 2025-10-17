package handlers

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/mascotmascot1/fileserver/internal/config"
)

// Handlers encapsulates the dependencies required by the HTTP handlers,
// such as the logger and configuration. This follows the dependency injection pattern,
// making the handlers easier to test and manage.
// Fields are unexported to prevent external packages from modifying their state after initialisation.
type Handlers struct {
	uploader *config.UploaderConfig
	logger   *log.Logger
}

// NewHandlers is a constructor that creates a new Handlers instance with the necessary dependencies.
func NewHandlers(cfg *config.Config, logger *log.Logger) *Handlers {
	return &Handlers{
		uploader: &cfg.Uploader,
		logger:   logger,
	}
}

// UploadHandler processes multipart/form-data requests to upload files.
func (h *Handlers) UploadHandler(w http.ResponseWriter, r *http.Request) {
	h.logger.Printf("received request from %s for %s\n", r.RemoteAddr, r.URL.Path)
	defer cleanupRequest(r)

	if r.Method != http.MethodPost {
		http.Error(w, "method must be POST", http.StatusMethodNotAllowed)
		return
	}

	// Why wrap the body? To prevent resource exhaustion. This enforces a hard limit
	// on the total request size, protecting the server from malicious or accidental DoS attacks.
	r.Body = http.MaxBytesReader(w, r.Body, h.uploader.GetMaxUploadSize())

	// Why parse with a memory limit? To balance performance against resource usage.
	// Form parts smaller than this limit are kept in RAM for speed; larger ones are
	// spooled to temporary files on disk, preventing a single request from consuming all memory.
	err := r.ParseMultipartForm(h.uploader.GetMaxFormMemSize())
	if err != nil {
		h.logger.Printf("error multipart parsing: %v\n", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	// Why MkdirAll? For idempotency and robustness. This ensures the storage path exists
	// without failing if it's already there, and it creates any necessary parent directories.
	err = os.MkdirAll(h.uploader.StorageDir, 0755) // Создаст все недостающие подкаталоги.
	if err != nil {
		h.logger.Printf("error creating file directory: %v\n", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	// Why open the root directory once? For security and performance.
	// It confines all subsequent file operations within this directory, preventing path traversal
	// attacks, and avoids the overhead of opening the directory repeatedly within the loop.
	root, err := os.OpenRoot(h.uploader.StorageDir)
	if err != nil {
		h.logger.Printf("error root opening: %v\n", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	defer root.Close()

	var uploadErrors []string
	// Process each file submitted in the form.
	for fieldName, fileHeaders := range r.MultipartForm.File {
		for _, fh := range fileHeaders {
			// Why can fh.Open fail? This operation deals with the client-provided data.
			// Failure here usually implies a client-side issue (e.g., malformed data)
			// or that the server's temporary file was cleaned up prematurely.
			file, err := fh.Open()
			if err != nil {
				msg := fmt.Sprintf("error getting file '%s' from field '%s'", fh.Filename, fieldName)
				h.logger.Printf("%s: %v\n", msg, err)
				uploadErrors = append(uploadErrors, msg)
				continue
			}

			// Why create the file with 'root.Create'? For security.
			// This guarantees the file is created inside the sandboxed storage directory.
			dst, err := root.Create(fh.Filename)
			if err != nil {
				// Failure here indicates a server-side problem (e.g., file permissions, disk space).
				msg := fmt.Sprintf("error creating file '%s'", fh.Filename)
				h.logger.Printf("%s: %v\n", msg, err)
				uploadErrors = append(uploadErrors, msg)
				file.Close() // Ensure the source file handle is closed on error.
				continue
			}

			// Why use a buffer for copying? To stream the file content efficiently
			// without loading the entire file into memory at once, which is crucial for large files.
			buf := make([]byte, 1<<20) // 1 MB buffer
			_, err = io.CopyBuffer(dst, file, buf)
			if err != nil {
				// An I/O error occurred whilst writing to the server's filesystem.
				msg := fmt.Sprintf("error writing file '%s'", fh.Filename)
				h.logger.Printf("%s: %v\n", msg, err)
				uploadErrors = append(uploadErrors, msg)

				// Ensure all opened resources for this file are closed on error.
				file.Close()
				dst.Close()

				// It's good practice to remove the partial file to avoid leaving corrupted data.
				if removeErr := os.Remove(filepath.Join(h.uploader.StorageDir, fh.Filename)); removeErr != nil {
					h.logger.Printf("failed to remove partial file '%s': %v", fh.Filename, removeErr)
				}
				continue
			}
			// Why close handles inside the loop? Using defer would leak file descriptors
			// until the handler returns, potentially exhausting system resources on requests with many files.
			file.Close()
			dst.Close()
		}
	}

	// Why check for upload errors? To provide clear feedback to the client
	// about which files, if any, failed to process.
	if len(uploadErrors) > 0 {
		errData, err := json.MarshalIndent(uploadErrors, "", "\t")
		if err != nil {
			h.logger.Printf("error marshalling uploadErrors to json: %v\n", err)
		}
		// Why StatusMultiStatus? It correctly signals that the request was partially
		// successful, as some files may have been saved whilst others failed.
		http.Error(w, string(errData), http.StatusMultiStatus)
		return
	}

	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)

	// After a successful status code, multiple writes to the response body are permissible.
	if _, err = w.Write([]byte("All files uploaded successfully\n")); err != nil {
		h.logger.Printf("error writing response: %s\n", err)
		return
	}
}

// DownloadHandle serves a specific file from the storage directory.
func (h *Handlers) DownloadHandle(w http.ResponseWriter, r *http.Request) {
	h.logger.Printf("received request from %s for %s\n", r.RemoteAddr, r.URL.Path)
	defer cleanupRequest(r)

	if r.Method != http.MethodGet {
		http.Error(w, "method must be GET", http.StatusMethodNotAllowed)
		return
	}

	const downloadPrefix = "/download/"
	fileName := strings.TrimPrefix(r.URL.Path, downloadPrefix)
	if fileName == "" {
		http.Error(w, "file name is not indicated", http.StatusBadRequest)
		return
	}

	// Why OpenRoot? For security. This ensures that the requested file path
	// is resolved strictly within the storage directory, preventing path traversal vulnerabilities.
	root, err := os.OpenRoot(h.uploader.StorageDir)
	if err != nil {
		// Failure here is an internal server error as the storage directory should be accessible.
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	defer root.Close()

	file, err := root.Open(fileName)
	if err != nil {
		// We assume the file doesn't exist if opening it fails.
		http.Error(w, "file is not found", http.StatusNotFound)
		return
	}
	defer file.Close()

	fileInfo, err := file.Stat()
	if err != nil {
		http.Error(w, "unable to access file", http.StatusInternalServerError)
		return
	}

	// Set headers to instruct the browser to download the file rather than displaying it.
	// Content-Length allows the browser to show download progress.
	w.Header().Set("Content-Length", fmt.Sprint(fileInfo.Size()))
	// application/octet-stream is a generic MIME type for binary data.
	w.Header().Set("Content-Type", "application/octet-stream")
	// Content-Disposition with 'attachment' suggests a "Save As" dialogue.
	// Why filepath.Base? For security, to sanitise the filename and prevent header injection attacks
	// where a malicious filename could manipulate the HTTP response.
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%s", filepath.Base(fileName)))
	// Explicitly write headers before the body. This is good practice as it finalises the response status.
	w.WriteHeader(http.StatusOK)

	_, err = io.Copy(w, file)
	if err != nil {
		h.logger.Printf("Error transferring file %s: %v", fileName, err)
		return
	}
}

// DownloadList serves a plain text file containing a list of all available files.
func (h *Handlers) DownloadList(w http.ResponseWriter, r *http.Request) {
	h.logger.Printf("received request from %s for %s\n", r.RemoteAddr, r.URL.Path)
	defer cleanupRequest(r)

	if r.Method != http.MethodGet {
		http.Error(w, "method must be GET", http.StatusMethodNotAllowed)
		return
	}

	files, err := os.ReadDir(h.uploader.StorageDir)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	// Why strings.Builder? To efficiently build the list in memory.
	// It's significantly more performant than repeated string concatenation (`+=`),
	// as it minimises memory allocations.
	var sb strings.Builder
	sb.WriteString("Files currently available:\n")
	for _, file := range files {
		sb.WriteString(file.Name())
		sb.WriteByte('\n')
	}
	fileList := sb.String()

	w.Header().Set("Content-Length", fmt.Sprint(len(fileList)))
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Header().Set("Content-Disposition", "attachment; filename=list.txt")
	w.WriteHeader(http.StatusOK)
	if _, err = w.Write([]byte(fileList)); err != nil {
		h.logger.Printf("error writing response: %s\n", err)
		return
	}
}

// Why have cleanupRequest? To ensure TCP connections can be reused (HTTP Keep-Alive).
// By reading and discarding the remainder of the request body, we ensure the connection
// is left in a clean state, ready for the next request.
func cleanupRequest(r *http.Request) {
	if r != nil {
		io.Copy(io.Discard, r.Body)
		r.Body.Close()
	}
}
