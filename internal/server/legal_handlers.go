package server

import (
	"net/http"
	"strings"
)

// handleGetLegalDocument returns the currently active legal document of a given type.
// Public endpoint: GET /api/legal/:type  (type = terms | disclaimer)
func (s *Server) handleGetLegalDocument(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		s.writeAPIError(w, "方法不允许", http.StatusMethodNotAllowed, "")
		return
	}
	if s.legalSvc == nil {
		s.writeAPISuccess(w, nil, "ok", http.StatusOK)
		return
	}

	docType := strings.TrimPrefix(r.URL.Path, "/api/legal/")
	if docType != "terms" && docType != "disclaimer" {
		s.writeAPIError(w, "无效的文档类型，支持: terms, disclaimer", http.StatusBadRequest, "")
		return
	}

	doc, err := s.legalSvc.GetActiveDocument(r.Context(), docType)
	if err != nil {
		s.writeAPISuccess(w, nil, "暂无内容", http.StatusOK)
		return
	}
	s.writeAPISuccess(w, doc, "ok", http.StatusOK)
}

// handleAdminLegalRouter routes GET/POST for /api/admin/legal
func (s *Server) handleAdminLegalRouter(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		s.handleAdminListLegal(w, r)
	case http.MethodPost:
		s.handleAdminSaveLegal(w, r)
	default:
		s.writeAPIError(w, "方法不允许", http.StatusMethodNotAllowed, "")
	}
}

// handleAdminListLegal returns all documents, optionally filtered by type.
func (s *Server) handleAdminListLegal(w http.ResponseWriter, r *http.Request) {
	if s.legalSvc == nil {
		s.writeAPIError(w, "服务未配置", http.StatusServiceUnavailable, "")
		return
	}

	docType := strings.TrimSpace(r.URL.Query().Get("type"))
	if docType == "" {
		docType = "terms"
	}
	if docType != "terms" && docType != "disclaimer" {
		s.writeAPIError(w, "无效的文档类型", http.StatusBadRequest, "")
		return
	}

	docs, err := s.legalSvc.ListDocuments(r.Context(), docType)
	if err != nil {
		s.writeAPIError(w, "获取文档列表失败: "+err.Error(), http.StatusInternalServerError, "")
		return
	}
	s.writeAPISuccess(w, docs, "ok", http.StatusOK)
}

// handleAdminSaveLegal creates a new version of a legal document (deactivates previous).
func (s *Server) handleAdminSaveLegal(w http.ResponseWriter, r *http.Request) {
	if s.legalSvc == nil {
		s.writeAPIError(w, "服务未配置", http.StatusServiceUnavailable, "")
		return
	}

	userID, ok := GetUserID(r.Context())
	if !ok {
		s.writeAPIError(w, "未认证", http.StatusUnauthorized, "")
		return
	}

	var req struct {
		Type    string `json:"type"`
		Title   string `json:"title"`
		Content string `json:"content"`
	}
	if err := parseJSONBody(r, &req); err != nil {
		s.writeAPIError(w, "无效的请求数据", http.StatusBadRequest, "")
		return
	}

	req.Type = strings.TrimSpace(req.Type)
	req.Title = strings.TrimSpace(req.Title)

	if req.Type != "terms" && req.Type != "disclaimer" {
		s.writeAPIError(w, "无效的文档类型，支持: terms, disclaimer", http.StatusBadRequest, "")
		return
	}
	if req.Title == "" {
		s.writeAPIError(w, "标题不能为空", http.StatusBadRequest, "")
		return
	}
	if req.Content == "" {
		s.writeAPIError(w, "内容不能为空", http.StatusBadRequest, "")
		return
	}

	doc, err := s.legalSvc.SaveDocument(r.Context(), req.Type, req.Title, req.Content, userID)
	if err != nil {
		s.writeAPIError(w, "保存失败: "+err.Error(), http.StatusInternalServerError, "")
		return
	}
	s.writeAPISuccess(w, doc, "保存成功", http.StatusCreated)
}
