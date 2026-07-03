package app

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"

	"webdown/internal/model"
)

const apiVersion = "0.2.0"

type apiResponse struct {
	OK      bool        `json:"ok"`
	Data    interface{} `json:"data,omitempty"`
	Error   string      `json:"error,omitempty"`
	Message string      `json:"message,omitempty"`
}

type healthInfo struct {
	Status        string         `json:"status"`
	APIVersion    string         `json:"api_version"`
	ServerTime    string         `json:"server_time"`
	UptimeSeconds int64          `json:"uptime_seconds"`
	UptimeHuman   string         `json:"uptime_human"`
	StartTime     string         `json:"start_time"`
	PublicBaseURL string         `json:"public_base_url"`
	Host          string         `json:"host"`
	Port          int            `json:"port"`
	MaxUploadMB   int64          `json:"max_upload_mb"`
	UploadDir     string         `json:"upload_dir"`
	Counts        map[string]int `json:"counts"`
	Total         int            `json:"total"`
}

type statsInfo struct {
	Total          int              `json:"total"`
	ByCategory     map[string]int   `json:"by_category"`
	TotalBytes     int64            `json:"total_bytes"`
	TotalSizeText  string           `json:"total_size_text"`
	Largest        *apkSummary      `json:"largest,omitempty"`
	Latest         *apkSummary      `json:"latest,omitempty"`
	ByCategorySize map[string]int64 `json:"by_category_bytes"`
}

type apkSummary struct {
	ID         string `json:"id"`
	AppName    string `json:"app_name"`
	Version    string `json:"version"`
	FileName   string `json:"file_name"`
	Category   string `json:"category"`
	Size       int64  `json:"size"`
	SizeText   string `json:"size_text"`
	UploadedAt string `json:"uploaded_at"`
	Download   string `json:"download_url"`
	Detail     string `json:"detail_url"`
	QR         string `json:"qr_url"`
}

type apkListResponse struct {
	Total    int          `json:"total"`
	Page     int          `json:"page"`
	PageSize int          `json:"page_size"`
	Category string       `json:"category"`
	Items    []apkSummary `json:"items"`
}

var serverStartTime = time.Now()

func (s *Server) apiHealth(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeAPIError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	now := time.Now()
	uptime := now.Sub(serverStartTime)
	counts := s.store.CountByCategory()
	total := 0
	for _, c := range counts {
		total += c
	}
	info := healthInfo{
		Status:        "ok",
		APIVersion:    apiVersion,
		ServerTime:    now.Format(time.RFC3339),
		UptimeSeconds: int64(uptime.Seconds()),
		UptimeHuman:   humanDuration(uptime),
		StartTime:     serverStartTime.Format(time.RFC3339),
		PublicBaseURL: s.publicBaseURL(r),
		Host:          s.cfg.Host,
		Port:          s.cfg.Port,
		MaxUploadMB:   s.cfg.MaxUploadMB,
		UploadDir:     s.cfg.UploadDir,
		Counts:        counts,
		Total:         total,
	}
	writeAPIOK(w, info)
}

func (s *Server) apiStats(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeAPIError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	items := s.store.List()
	counts := map[string]int{
		model.CategoryAndroid: 0,
		model.CategoryIOS:     0,
		model.CategoryFile:    0,
	}
	bySize := map[string]int64{
		model.CategoryAndroid: 0,
		model.CategoryIOS:     0,
		model.CategoryFile:    0,
	}
	var total int64
	var largest *apkSummary
	var latest *apkSummary

	for _, it := range items {
		cat := it.ResolvedCategory()
		counts[cat]++
		bySize[cat] += it.Size
		total += it.Size

		summary := s.summaryFor(r, it)
		if largest == nil || it.Size > largest.Size {
			summaryCopy := summary
			largest = &summaryCopy
		}
		if latest == nil || it.UploadedAt.After(parseTimeOrZero(latest.UploadedAt)) {
			summaryCopy := summary
			latest = &summaryCopy
		}
	}

	totalCount := counts[model.CategoryAndroid] + counts[model.CategoryIOS] + counts[model.CategoryFile]
	stats := statsInfo{
		Total:          totalCount,
		ByCategory:     counts,
		TotalBytes:     total,
		TotalSizeText:  formatBytes(total),
		Largest:        largest,
		Latest:         latest,
		ByCategorySize: bySize,
	}
	writeAPIOK(w, stats)
}

func (s *Server) apiList(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeAPIError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	q := r.URL.Query()
	category := normalizeCategory(q.Get("category"))
	page, _ := strconv.Atoi(q.Get("page"))
	if page < 1 {
		page = 1
	}
	pageSize, _ := strconv.Atoi(q.Get("page_size"))
	if pageSize < 1 || pageSize > 200 {
		pageSize = 50
	}

	items := s.store.ListByCategory(category)
	total := len(items)

	start := (page - 1) * pageSize
	if start > total {
		start = total
	}
	end := start + pageSize
	if end > total {
		end = total
	}

	summaries := make([]apkSummary, 0, end-start)
	for _, it := range items[start:end] {
		summaries = append(summaries, s.summaryFor(r, it))
	}

	writeAPIOK(w, apkListResponse{
		Total:    total,
		Page:     page,
		PageSize: pageSize,
		Category: category,
		Items:    summaries,
	})
}

func (s *Server) apiInfo(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeAPIError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	id := strings.TrimPrefix(r.URL.Path, "/api/info/")
	apk, ok := s.store.Get(id)
	if !ok {
		writeAPIError(w, http.StatusNotFound, "package not found")
		return
	}
	writeAPIOK(w, s.summaryFor(r, apk))
}

func (s *Server) apiCheck(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeAPIError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	id := strings.TrimPrefix(r.URL.Path, "/api/check/")
	apk, ok := s.store.Get(id)
	if !ok {
		writeAPIOK(w, map[string]interface{}{
			"id":     id,
			"exists": false,
		})
		return
	}
	writeAPIOK(w, map[string]interface{}{
		"id":           id,
		"exists":       true,
		"category":     apk.ResolvedCategory(),
		"file_name":    apk.FileName,
		"size":         apk.Size,
		"size_text":    formatBytes(apk.Size),
		"uploaded_at":  apk.UploadedAt.Format(time.RFC3339),
		"download_url": s.downloadURL(r, apk.ID),
	})
}

func (s *Server) summaryFor(r *http.Request, it model.APK) apkSummary {
	return apkSummary{
		ID:         it.ID,
		AppName:    it.AppName,
		Version:    it.Version,
		FileName:   it.FileName,
		Category:   it.ResolvedCategory(),
		Size:       it.Size,
		SizeText:   formatBytes(it.Size),
		UploadedAt: it.UploadedAt.Format(time.RFC3339),
		Download:   s.downloadURL(r, it.ID),
		Detail:     "/apk/" + it.ID,
		QR:         "/qr/" + it.ID + ".png?size=320",
	}
}

func writeAPIOK(w http.ResponseWriter, data interface{}) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	_ = json.NewEncoder(w).Encode(apiResponse{OK: true, Data: data})
}

func writeAPIError(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(apiResponse{OK: false, Error: message})
}

func humanDuration(d time.Duration) string {
	seconds := int64(d.Seconds())
	if seconds < 60 {
		return strconv.FormatInt(seconds, 10) + "s"
	}
	minutes := seconds / 60
	if minutes < 60 {
		return strconv.FormatInt(minutes, 10) + "m" + strconv.FormatInt(seconds%60, 10) + "s"
	}
	hours := minutes / 60
	if hours < 24 {
		return strconv.FormatInt(hours, 10) + "h" + strconv.FormatInt(minutes%60, 10) + "m"
	}
	days := hours / 24
	return strconv.FormatInt(days, 10) + "d" + strconv.FormatInt(hours%24, 10) + "h"
}

func parseTimeOrZero(s string) time.Time {
	if t, err := time.Parse(time.RFC3339, s); err == nil {
		return t
	}
	return time.Time{}
}
