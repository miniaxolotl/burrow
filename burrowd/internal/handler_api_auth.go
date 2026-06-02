package internal

import (
	"encoding/json"
	"net/http"
)

type registerRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type registerResponse struct {
	User         *User  `json:"user"`
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
}

type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type loginResponse struct {
	User         *User  `json:"user"`
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
}

type refreshRequest struct {
	RefreshToken string `json:"refresh_token"`
}

type refreshResponse struct {
	AccessToken string `json:"access_token"`
}

type meResponse struct {
	ID    int64  `json:"id"`
	UUID  string `json:"uuid"`
	Email string `json:"email"`
	Role  string `json:"role"`
	Plan  string `json:"plan"`
}

func (s *Server) handleRegister(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		jsonError(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req registerRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, "Invalid request", http.StatusBadRequest)
		return
	}
	if req.Email == "" || req.Password == "" {
		jsonError(w, "Email and password required", http.StatusBadRequest)
		return
	}
	user, access, refresh, err := Register(r.Context(), s.pg, s.jwtSecret, req.Email, req.Password)
	if err != nil {
		jsonError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	respond(w, http.StatusCreated, registerResponse{User: user, AccessToken: access, RefreshToken: refresh})
}

func (s *Server) handleLogin(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		jsonError(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req loginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, "Invalid request", http.StatusBadRequest)
		return
	}
	if req.Email == "" || req.Password == "" {
		jsonError(w, "Email and password required", http.StatusBadRequest)
		return
	}
	user, access, refresh, err := Login(r.Context(), s.pg, s.jwtSecret, req.Email, req.Password)
	if err != nil {
		jsonError(w, "Invalid credentials", http.StatusUnauthorized)
		return
	}
	respond(w, http.StatusOK, loginResponse{User: user, AccessToken: access, RefreshToken: refresh})
}

func (s *Server) handleRefreshToken(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		jsonError(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req refreshRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, "Invalid request", http.StatusBadRequest)
		return
	}
	if req.RefreshToken == "" {
		jsonError(w, "Refresh token required", http.StatusBadRequest)
		return
	}
	access, err := RefreshToken(r.Context(), s.pg, s.jwtSecret, req.RefreshToken)
	if err != nil {
		jsonError(w, "Invalid refresh token", http.StatusUnauthorized)
		return
	}
	respond(w, http.StatusOK, refreshResponse{AccessToken: access})
}

func (s *Server) handleLogout(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		jsonError(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req refreshRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, "Invalid request", http.StatusBadRequest)
		return
	}
	if err := Logout(r.Context(), s.pg, req.RefreshToken); err != nil {
		jsonError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleMe(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		jsonError(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	userID := userIDFromContext(r.Context())
	if userID == nil {
		jsonError(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	user, err := s.pg.GetUserByID(r.Context(), *userID)
	if err != nil {
		jsonError(w, "User not found", http.StatusNotFound)
		return
	}
	respond(w, http.StatusOK, meResponse{
		ID:    user.ID,
		UUID:  user.UUID,
		Email: user.Email,
		Role:  user.Role,
		Plan:  user.Plan,
	})
}

func respond(w http.ResponseWriter, code int, payload interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(payload)
}
