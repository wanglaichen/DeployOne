package model

import "time"

const (
	CategoryAndroid = "android"
	CategoryIOS     = "ios"
	CategoryFile    = "file"
)

type APK struct {
	ID          string    `json:"id"`
	Category    string    `json:"category"`
	AppName     string    `json:"app_name"`
	Version     string    `json:"version"`
	Description string    `json:"description"`
	FileName    string    `json:"file_name"`
	StoredName  string    `json:"stored_name"`
	Size        int64     `json:"size"`
	UploadedAt  time.Time `json:"uploaded_at"`
}

func (a APK) ResolvedCategory() string {
	switch a.Category {
	case CategoryIOS, CategoryFile:
		return a.Category
	default:
		return CategoryAndroid
	}
}
