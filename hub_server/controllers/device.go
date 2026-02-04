package controllers

import (
	"encoding/json"
	"net/http"

	"wx_channel/hub_server/database"
	"wx_channel/hub_server/services"
)

// GenerateBindToken returns a short code for the user to input in the client
func GenerateBindToken(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("user_id").(uint)

	token, err := services.Binder.GenerateToken(userID)
	if err != nil {
		http.Error(w, "Failed to generate token", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"token": token,
	})
}

// GetUserDevices returns all devices bound to the current user
func GetUserDevices(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("user_id").(uint)

	user, err := database.GetUserByID(userID)
	if err != nil {
		http.Error(w, "User not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(user.Devices)
}

// UnbindDevice removes the binding between a device and the user
func UnbindDevice(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("user_id").(uint)

	// 解析请求参数
	var req struct {
		DeviceID string `json:"device_id"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.DeviceID == "" {
		http.Error(w, "device_id is required", http.StatusBadRequest)
		return
	}

	// 获取设备信息
	node, err := database.GetNodeByID(req.DeviceID)
	if err != nil {
		http.Error(w, "Device not found", http.StatusNotFound)
		return
	}

	// 检查设备是否属于当前用户
	if node.UserID != userID {
		http.Error(w, "Permission denied", http.StatusForbidden)
		return
	}

	// 解绑设备
	if err := database.UnbindNode(req.DeviceID); err != nil {
		http.Error(w, "Failed to unbind device", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": "Device unbound successfully",
	})
}

// DeleteDevice permanently deletes a device from the database
func DeleteDevice(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("user_id").(uint)

	// 解析请求参数
	var req struct {
		DeviceID string `json:"device_id"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.DeviceID == "" {
		http.Error(w, "device_id is required", http.StatusBadRequest)
		return
	}

	// 获取设备信息
	node, err := database.GetNodeByID(req.DeviceID)
	if err != nil {
		http.Error(w, "Device not found", http.StatusNotFound)
		return
	}

	// 检查设备是否属于当前用户
	if node.UserID != userID {
		http.Error(w, "Permission denied", http.StatusForbidden)
		return
	}

	// 删除设备
	if err := database.DeleteNode(req.DeviceID); err != nil {
		http.Error(w, "Failed to delete device", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": "Device deleted successfully",
	})
}
