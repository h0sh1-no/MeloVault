package server

import (
	"encoding/json"
	"io"
	"net/http"
)

func (s *Server) handleSettingsRouter(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		s.handleGetSettings(w, r)
	case http.MethodPut:
		s.handleUpdateSettings(w, r)
	default:
		s.writeAPIError(w, "方法不允许", http.StatusMethodNotAllowed, "")
	}
}

func (s *Server) handleGetSettings(w http.ResponseWriter, r *http.Request) {
	userID, ok := GetUserID(r.Context())
	if !ok {
		s.writeAPIError(w, "未认证", http.StatusUnauthorized, "")
		return
	}

	us, err := s.settingsSvc.Get(r.Context(), userID)
	if err != nil {
		s.writeAPIError(w, "获取设置失败", http.StatusInternalServerError, "")
		return
	}

	s.writeAPISuccess(w, us.Settings, "获取设置成功", http.StatusOK)
}

func (s *Server) handleUpdateSettings(w http.ResponseWriter, r *http.Request) {
	userID, ok := GetUserID(r.Context())
	if !ok {
		s.writeAPIError(w, "未认证", http.StatusUnauthorized, "")
		return
	}

	body, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
	if err != nil {
		s.writeAPIError(w, "无效的请求数据", http.StatusBadRequest, "")
		return
	}

	var partial map[string]any
	if err := json.Unmarshal(body, &partial); err != nil {
		s.writeAPIError(w, "无效的JSON数据", http.StatusBadRequest, "")
		return
	}

	us, err := s.settingsSvc.Update(r.Context(), userID, partial)
	if err != nil {
		s.writeAPIError(w, "更新设置失败: "+err.Error(), http.StatusInternalServerError, "")
		return
	}

	s.writeAPISuccess(w, us.Settings, "设置已保存", http.StatusOK)
}
