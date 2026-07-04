package app

import (
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path/filepath"

	"webdown/internal/model"
)

type AdminLoginData struct {
	Error string
}

type AdminData struct {
	Items       []APKView
	ActiveTab   string
	Tabs        []TabView
	Error       string
	Success     string
	TotalCount  int
}

func (s *Server) adminLogin(w http.ResponseWriter, r *http.Request) {
	if s.isAdmin(r) {
		http.Redirect(w, r, "/admin", http.StatusSeeOther)
		return
	}

	switch r.Method {
	case http.MethodGet:
		data := AdminLoginData{Error: r.URL.Query().Get("error")}
		if err := s.templates.ExecuteTemplate(w, "admin_login.html", data); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	case http.MethodPost:
		if err := r.ParseForm(); err != nil {
			http.Redirect(w, r, "/admin/login?error="+url.QueryEscape("登录失败"), http.StatusSeeOther)
			return
		}
		username := r.FormValue("username")
		password := r.FormValue("password")
		if username != s.cfg.AdminUser || password != s.cfg.AdminPassword {
			http.Redirect(w, r, "/admin/login?error="+url.QueryEscape("用户名或密码错误"), http.StatusSeeOther)
			return
		}
		if err := s.setAdminSession(w); err != nil {
			http.Error(w, "create session failed", http.StatusInternalServerError)
			return
		}
		http.Redirect(w, r, "/admin", http.StatusSeeOther)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *Server) adminLogout(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	s.clearAdminSession(w, r)
	http.Redirect(w, r, "/admin/login", http.StatusSeeOther)
}

func (s *Server) adminIndex(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if !s.requireAdmin(w, r) {
		return
	}

	activeTab := normalizeCategory(r.URL.Query().Get("tab"))
	if r.URL.Query().Get("sync") == "1" {
		removed, err := s.syncMissingFiles()
		if err != nil {
			http.Redirect(w, r, "/admin?tab="+activeTab+"&error="+url.QueryEscape("同步失败"), http.StatusSeeOther)
			return
		}
		msg := "列表已刷新"
		if removed > 0 {
			msg = fmt.Sprintf("列表已刷新，移除了 %d 个失效记录", removed)
		}
		http.Redirect(w, r, "/admin?tab="+activeTab+"&success="+url.QueryEscape(msg), http.StatusSeeOther)
		return
	}

	counts := s.store.CountByCategory()
	items := s.viewsByCategory(r, activeTab)
	total := counts[model.CategoryAndroid] + counts[model.CategoryIOS] + counts[model.CategoryFile]

	data := AdminData{
		Items:      items,
		ActiveTab:  activeTab,
		TotalCount: total,
		Error:      r.URL.Query().Get("error"),
		Success:    r.URL.Query().Get("success"),
		Tabs: []TabView{
			{Key: model.CategoryAndroid, Label: "安卓包", Desc: "APK 安装包", Icon: iconAndroidSVG, Count: counts[model.CategoryAndroid], Active: activeTab == model.CategoryAndroid},
			{Key: model.CategoryIOS, Label: "苹果包", Desc: "IPA 测试包", Icon: iconAppleSVG, Count: counts[model.CategoryIOS], Active: activeTab == model.CategoryIOS},
			{Key: model.CategoryFile, Label: "普通文件", Desc: "任意格式分享", Icon: iconFolderSVG, Count: counts[model.CategoryFile], Active: activeTab == model.CategoryFile},
		},
	}
	if err := s.templates.ExecuteTemplate(w, "admin.html", data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (s *Server) adminDelete(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if !s.requireAdmin(w, r) {
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Redirect(w, r, "/admin?error="+url.QueryEscape("删除失败"), http.StatusSeeOther)
		return
	}

	id := r.FormValue("id")
	tab := normalizeCategory(r.FormValue("tab"))
	item, ok, err := s.store.Delete(id)
	if err != nil {
		http.Redirect(w, r, "/admin?tab="+tab+"&error="+url.QueryEscape("删除失败"), http.StatusSeeOther)
		return
	}
	if !ok {
		http.Redirect(w, r, "/admin?tab="+tab+"&error="+url.QueryEscape("文件不存在"), http.StatusSeeOther)
		return
	}

	filePath := filepath.Join(s.cfg.UploadDir, item.StoredName)
	_ = os.Remove(filePath)

	http.Redirect(w, r, "/admin?tab="+tab+"&success="+url.QueryEscape("已删除："+item.AppName), http.StatusSeeOther)
}

func (s *Server) adminDeleteAll(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if !s.requireAdmin(w, r) {
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Redirect(w, r, "/admin?error="+url.QueryEscape("删除失败"), http.StatusSeeOther)
		return
	}

	tab := normalizeCategory(r.FormValue("tab"))
	scope := r.FormValue("scope")
	category := ""
	message := "已清空全部文件"
	switch scope {
	case "category":
		category = tab
		message = "已清空当前分类下的全部文件"
	case "all":
		category = ""
		message = "已清空全部文件"
	default:
		http.Redirect(w, r, "/admin?tab="+tab+"&error="+url.QueryEscape("无效的删除范围"), http.StatusSeeOther)
		return
	}

	removed, err := s.store.DeleteByCategory(category)
	if err != nil {
		http.Redirect(w, r, "/admin?tab="+tab+"&error="+url.QueryEscape("删除失败"), http.StatusSeeOther)
		return
	}
	for _, item := range removed {
		filePath := filepath.Join(s.cfg.UploadDir, item.StoredName)
		_ = os.Remove(filePath)
	}
	if len(removed) > 0 {
		message = fmt.Sprintf("%s（%d 个）", message, len(removed))
	} else {
		message = "没有可删除的文件"
	}
	http.Redirect(w, r, "/admin?tab="+tab+"&success="+url.QueryEscape(message), http.StatusSeeOther)
}

func (s *Server) apiSync(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost && r.Method != http.MethodGet {
		writeAPIError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	removed, err := s.syncMissingFiles()
	if err != nil {
		writeAPIError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeAPIOK(w, map[string]int{"removed": removed})
}

func (s *Server) syncMissingFiles() (int, error) {
	return s.store.PruneMissing(s.cfg.UploadDir)
}
