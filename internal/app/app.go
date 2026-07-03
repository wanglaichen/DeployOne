package app

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"html/template"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	qrcode "github.com/skip2/go-qrcode"

	"webdown/internal/config"
	"webdown/internal/model"
	"webdown/internal/store"
)

type Server struct {
	cfg       config.Config
	store     *store.Store
	templates *template.Template
}

type TabView struct {
	Key    string
	Label  string
	Desc   string
	Icon   string
	Count  int
	Active bool
}

type IndexData struct {
	Items         []APKView
	PublicURL     string
	MaxUploadMB   int64
	Error         string
	Success       string
	ActiveTab     string
	Tabs          []TabView
	PageTitle     string
	HeroEyebrow   string
	HeroTitle     string
	HeroAccent    string
	PageDesc      string
	UploadTitle   string
	FileLabel     string
	ListTitle     string
	EmptyText     string
	SubmitText    string
	Accept        string
	ShowQR        bool
}

type DetailData struct {
	Item          APKView
	PublicURL     string
	ActiveTab     string
	PageTitle     string
	PageDesc      string
	DownloadLabel string
	QRLabel       string
	BackURL       string
}

type APKView struct {
	model.APK
	SizeText     string
	UploadedAt   string
	DownloadURL  string
	QRURL        string
	LargeQRURL   string
	DetailURL    string
	CategoryText string
}

func New(cfg config.Config, st *store.Store) (*Server, error) {
	if err := os.MkdirAll(cfg.UploadDir, 0755); err != nil {
		return nil, fmt.Errorf("create upload dir: %w", err)
	}

	tpl, err := template.New("").ParseGlob(filepath.Join("web", "templates", "*.html"))
	if err != nil {
		return nil, fmt.Errorf("parse templates: %w", err)
	}

	return &Server{cfg: cfg, store: st, templates: tpl}, nil
}

func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/", s.index)
	mux.HandleFunc("/apk/", s.detail)
	mux.HandleFunc("/upload", s.upload)
	mux.HandleFunc("/download/", s.download)
	mux.HandleFunc("/qr/", s.qr)
	mux.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir(filepath.Join("web", "static")))))
	return mux
}

func (s *Server) index(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}

	activeTab := normalizeCategory(r.URL.Query().Get("tab"))
	meta := getCategoryMeta(activeTab)
	counts := s.store.CountByCategory()

	data := IndexData{
		Items:       s.viewsByCategory(r, activeTab),
		PublicURL:   s.publicBaseURL(r),
		MaxUploadMB: s.cfg.MaxUploadMB,
		Error:       r.URL.Query().Get("error"),
		Success:     r.URL.Query().Get("success"),
		ActiveTab:   activeTab,
		Tabs: []TabView{
			{Key: model.CategoryAndroid, Label: "安卓包", Desc: "APK 安装包", Icon: "🤖", Count: counts[model.CategoryAndroid], Active: activeTab == model.CategoryAndroid},
			{Key: model.CategoryIOS, Label: "苹果包", Desc: "IPA 测试包", Icon: "🍎", Count: counts[model.CategoryIOS], Active: activeTab == model.CategoryIOS},
			{Key: model.CategoryFile, Label: "普通文件", Desc: "任意格式分享", Icon: "📁", Count: counts[model.CategoryFile], Active: activeTab == model.CategoryFile},
		},
		PageTitle:   meta.PageTitle,
		HeroEyebrow: meta.HeroEyebrow,
		HeroTitle:   meta.HeroTitle,
		HeroAccent:  meta.HeroAccent,
		PageDesc:    meta.PageDesc,
		UploadTitle: meta.UploadTitle,
		FileLabel:   meta.FileLabel,
		ListTitle:   meta.ListTitle,
		EmptyText:   meta.EmptyText,
		SubmitText:  meta.SubmitText,
		Accept:      meta.Accept,
		ShowQR:      meta.ShowQR,
	}
	if err := s.templates.ExecuteTemplate(w, "index.html", data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (s *Server) detail(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	id := strings.TrimPrefix(r.URL.Path, "/apk/")
	apk, ok := s.store.Get(id)
	if !ok {
		http.NotFound(w, r)
		return
	}

	activeTab := apk.ResolvedCategory()
	meta := getCategoryMeta(activeTab)
	data := DetailData{
		Item:          s.view(r, apk),
		PublicURL:     s.publicBaseURL(r),
		ActiveTab:     activeTab,
		PageTitle:     meta.DetailTitle,
		PageDesc:      meta.DetailDesc,
		DownloadLabel: meta.DownloadLabel,
		QRLabel:       meta.QRLabel,
		BackURL:       "/?tab=" + activeTab,
	}
	if err := s.templates.ExecuteTemplate(w, "detail.html", data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (s *Server) upload(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	category := normalizeCategory(r.FormValue("category"))
	meta := getCategoryMeta(category)

	maxBytes := s.cfg.MaxUploadMB * 1024 * 1024
	r.Body = http.MaxBytesReader(w, r.Body, maxBytes)

	if err := r.ParseMultipartForm(maxBytes); err != nil {
		s.redirectWithMessage(w, r, category, "error", "上传失败，文件大小可能超过限制")
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		file, header, err = r.FormFile("apk")
	}
	if err != nil {
		s.redirectWithMessage(w, r, category, "error", meta.SelectError)
		return
	}
	defer file.Close()

	ext := strings.ToLower(filepath.Ext(header.Filename))
	if !isAllowedExtension(category, ext) {
		s.redirectWithMessage(w, r, category, "error", meta.TypeError)
		return
	}

	id, err := newID()
	if err != nil {
		s.redirectWithMessage(w, r, category, "error", "生成文件 ID 失败")
		return
	}

	originalName := safeFileName(header.Filename, ext)
	storedName := id + ext
	dstPath := filepath.Join(s.cfg.UploadDir, storedName)
	dst, err := os.Create(dstPath)
	if err != nil {
		s.redirectWithMessage(w, r, category, "error", "创建上传文件失败")
		return
	}
	size, copyErr := io.Copy(dst, file)
	closeErr := dst.Close()
	if copyErr != nil || closeErr != nil {
		_ = os.Remove(dstPath)
		s.redirectWithMessage(w, r, category, "error", "保存上传文件失败")
		return
	}

	apk := model.APK{
		ID:         id,
		Category:   category,
		AppName:    strings.TrimSpace(r.FormValue("app_name")),
		Version:    strings.TrimSpace(r.FormValue("version")),
		FileName:   originalName,
		StoredName: storedName,
		Size:       size,
		UploadedAt: time.Now(),
	}
	if apk.AppName == "" {
		apk.AppName = strings.TrimSuffix(originalName, ext)
	}

	if err := s.store.Add(apk); err != nil {
		_ = os.Remove(dstPath)
		s.redirectWithMessage(w, r, category, "error", "保存文件信息失败")
		return
	}

	s.redirectWithMessage(w, r, category, "success", "上传成功")
}

func (s *Server) download(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	id := strings.TrimPrefix(r.URL.Path, "/download/")
	apk, ok := s.store.Get(id)
	if !ok {
		http.NotFound(w, r)
		return
	}

	filePath := filepath.Join(s.cfg.UploadDir, apk.StoredName)
	w.Header().Set("Content-Type", contentTypeFor(apk))
	w.Header().Set("Content-Disposition", "attachment; filename*=UTF-8''"+url.QueryEscape(apk.FileName))
	http.ServeFile(w, r, filePath)
}

func (s *Server) qr(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	id := strings.TrimPrefix(r.URL.Path, "/qr/")
	id = strings.TrimSuffix(id, ".png")
	if _, ok := s.store.Get(id); !ok {
		http.NotFound(w, r)
		return
	}

	png, err := qrcode.Encode(s.downloadURL(r, id), qrcode.Medium, qrSize(r))
	if err != nil {
		http.Error(w, "生成二维码失败", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "image/png")
	w.Header().Set("Cache-Control", "no-store")
	_, _ = w.Write(png)
}

func (s *Server) viewsByCategory(r *http.Request, category string) []APKView {
	items := s.store.ListByCategory(category)
	views := make([]APKView, 0, len(items))
	for _, item := range items {
		views = append(views, s.view(r, item))
	}
	return views
}

func (s *Server) view(r *http.Request, item model.APK) APKView {
	return APKView{
		APK:          item,
		SizeText:     formatBytes(item.Size),
		UploadedAt:   item.UploadedAt.Format("2006-01-02 15:04:05"),
		DownloadURL:  s.downloadURL(r, item.ID),
		QRURL:        "/qr/" + item.ID + ".png?size=320",
		LargeQRURL:   "/qr/" + item.ID + ".png?size=520",
		DetailURL:    "/apk/" + item.ID,
		CategoryText: categoryLabel(item.ResolvedCategory()),
	}
}

func (s *Server) downloadURL(r *http.Request, id string) string {
	return strings.TrimRight(s.publicBaseURL(r), "/") + "/download/" + id
}

func (s *Server) publicBaseURL(r *http.Request) string {
	if s.cfg.PublicBaseURL != "" {
		return s.cfg.PublicBaseURL
	}

	scheme := "http"
	if r.TLS != nil {
		scheme = "https"
	}
	return scheme + "://" + r.Host
}

func (s *Server) redirectWithMessage(w http.ResponseWriter, r *http.Request, category string, key string, message string) {
	target := "/?tab=" + normalizeCategory(category) + "&" + key + "=" + url.QueryEscape(message)
	http.Redirect(w, r, target, http.StatusSeeOther)
}

type categoryMeta struct {
	PageTitle     string
	HeroEyebrow   string
	HeroTitle     string
	HeroAccent    string
	PageDesc      string
	UploadTitle   string
	FileLabel     string
	ListTitle     string
	EmptyText     string
	SubmitText    string
	Accept        string
	SelectError   string
	TypeError     string
	DetailTitle   string
	DetailDesc    string
	DownloadLabel string
	QRLabel       string
	ShowQR        bool
}

func getCategoryMeta(category string) categoryMeta {
	switch normalizeCategory(category) {
	case model.CategoryIOS:
		return categoryMeta{
			PageTitle:     "WebDown · 苹果安装包",
			HeroEyebrow:   "iOS Package",
			HeroTitle:     "苹果安装包",
			HeroAccent:    "分发中心",
			PageDesc:      "上传 IPA 测试包，自动生成下载链接与二维码，扫码即可获取安装文件。",
			UploadTitle:   "上传苹果包",
			FileLabel:     "IPA 文件",
			ListTitle:     "苹果包列表",
			EmptyText:     "还没有上传苹果包，请先上传一个 IPA 文件。",
			SubmitText:    "上传 IPA",
			Accept:        ".ipa",
			SelectError:   "请选择 IPA 文件",
			TypeError:     "只允许上传 .ipa 文件",
			DetailTitle:   "苹果包详情",
			DetailDesc:    "此页面显示当前苹果包的完整下载地址和大二维码。",
			DownloadLabel: "下载 IPA",
			QRLabel:       "扫码下载 IPA",
			ShowQR:        true,
		}
	case model.CategoryFile:
		return categoryMeta{
			PageTitle:     "WebDown · 文件分享",
			HeroEyebrow:   "File Sharing",
			HeroTitle:     "文件",
			HeroAccent:    "快传中心",
			PageDesc:      "上传任意文件，即时生成分享链接与二维码，团队取用更方便。",
			UploadTitle:   "上传普通文件",
			FileLabel:     "文件",
			ListTitle:     "文件列表",
			EmptyText:     "还没有上传文件，请先上传一个文件。",
			SubmitText:    "上传文件",
			Accept:        "*/*",
			SelectError:   "请选择要上传的文件",
			TypeError:     "请上传有效的文件",
			DetailTitle:   "文件详情",
			DetailDesc:    "此页面显示当前文件的完整下载地址和大二维码。",
			DownloadLabel: "下载文件",
			QRLabel:       "扫码下载文件",
			ShowQR:        true,
		}
	default:
		return categoryMeta{
			PageTitle:     "WebDown · 安卓安装包",
			HeroEyebrow:   "Android Package",
			HeroTitle:     "安卓安装包",
			HeroAccent:    "分发中心",
			PageDesc:      "上传 APK 安装包，自动生成下载链接与二维码，扫码即可安装测试。",
			UploadTitle:   "上传安卓包",
			FileLabel:     "APK 文件",
			ListTitle:     "安卓包列表",
			EmptyText:     "还没有上传安卓包，请先上传一个 APK 文件。",
			SubmitText:    "上传 APK",
			Accept:        ".apk,application/vnd.android.package-archive",
			SelectError:   "请选择 APK 文件",
			TypeError:     "只允许上传 .apk 文件",
			DetailTitle:   "安卓包详情",
			DetailDesc:    "此页面显示当前安卓包的完整下载地址和大二维码。",
			DownloadLabel: "下载 APK",
			QRLabel:       "扫码下载 APK",
			ShowQR:        true,
		}
	}
}

func normalizeCategory(category string) string {
	switch strings.ToLower(strings.TrimSpace(category)) {
	case model.CategoryIOS:
		return model.CategoryIOS
	case model.CategoryFile:
		return model.CategoryFile
	default:
		return model.CategoryAndroid
	}
}

func categoryLabel(category string) string {
	switch normalizeCategory(category) {
	case model.CategoryIOS:
		return "苹果包"
	case model.CategoryFile:
		return "普通文件"
	default:
		return "安卓包"
	}
}

func isAllowedExtension(category string, ext string) bool {
	switch normalizeCategory(category) {
	case model.CategoryIOS:
		return ext == ".ipa"
	case model.CategoryFile:
		return ext != ""
	default:
		return ext == ".apk"
	}
}

func contentTypeFor(item model.APK) string {
	switch strings.ToLower(filepath.Ext(item.FileName)) {
	case ".apk":
		return "application/vnd.android.package-archive"
	case ".ipa":
		return "application/octet-stream"
	default:
		return "application/octet-stream"
	}
}

func newID() (string, error) {
	var b [8]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", err
	}
	return hex.EncodeToString(b[:]), nil
}

func safeFileName(name string, ext string) string {
	name = filepath.Base(name)
	name = strings.TrimSpace(name)
	if name == "" || name == "." || name == string(filepath.Separator) {
		if ext == "" {
			return "download.bin"
		}
		return "download" + ext
	}
	return name
}

func qrSize(r *http.Request) int {
	size, err := strconv.Atoi(r.URL.Query().Get("size"))
	if err != nil {
		return 320
	}
	if size < 160 {
		return 160
	}
	if size > 800 {
		return 800
	}
	return size
}

func formatBytes(size int64) string {
	const unit = 1024
	if size < unit {
		return strconv.FormatInt(size, 10) + " B"
	}
	div, exp := int64(unit), 0
	for n := size / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %ciB", float64(size)/float64(div), "KMGTPE"[exp])
}
