package middleware

import "net/http"

type WrapperWriter struct {
	http.ResponseWriter
	StatusCode  int
	Bytes       int
	wroteHeader bool
}

func (w *WrapperWriter) WriteHeader(statusCode int) {
	if w.wroteHeader {
		return
	}

	w.StatusCode = statusCode
	w.wroteHeader = true
	w.ResponseWriter.WriteHeader(statusCode)
}

func (w *WrapperWriter) Write(b []byte) (int, error) {
	if !w.wroteHeader {
		w.WriteHeader(http.StatusOK)
	}

	n, err := w.ResponseWriter.Write(b)
	w.Bytes += n

	return n, err
}
