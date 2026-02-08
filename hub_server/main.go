package main

import (
	"log"
	"net/http"
	"os"
	"runtime/debug"
	"strings"

	"wx_channel/hub_server/controllers"
	"wx_channel/hub_server/database"
	"wx_channel/hub_server/middleware"
	"wx_channel/hub_server/services"
	"wx_channel/hub_server/ws"

	"github.com/gorilla/mux"
)

func main() {
	if err := middleware.InitJWTSecretFromEnv(); err != nil {
		log.Fatalf("Invalid JWT secret configuration: %v", err)
	}

	// 1. 初始化数据库
	if err := database.InitDB("hub_server.db"); err != nil {
		log.Fatalf("Failed to init database: %v", err)
	}

	// 2. 初始化 WebSocket Hub
	hub := ws.NewHub()
	go hub.Run()

	// 2.5 启动积分矿工服务 (在线时长统计)
	services.StartMiningService()

	// 3. Middleware: Panic Recovery
	withRecovery := func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if err := recover(); err != nil {
					log.Printf("PANIC: %v\nStack: %s", err, string(debug.Stack()))
					http.Error(w, "Internal Server Error", 500)
				}
			}()
			next(w, r)
		}
	}

	// 4. 创建路由器
	router := mux.NewRouter()

	// WebSocket 接入点
	router.HandleFunc("/ws/client", withRecovery(hub.ServeWs)).Methods("GET")

	// Auth API
	router.HandleFunc("/api/auth/register", withRecovery(controllers.Register)).Methods("POST")
	router.HandleFunc("/api/auth/login", withRecovery(controllers.Login)).Methods("POST")

	// Protected API (Need Auth)
	router.HandleFunc("/api/auth/profile", withRecovery(middleware.AuthRequired(controllers.GetProfile))).Methods("GET")
	router.HandleFunc("/api/user/change-password", withRecovery(middleware.AuthRequired(controllers.ChangePassword))).Methods("POST")
	router.HandleFunc("/api/user/stats", withRecovery(middleware.AuthRequired(controllers.GetUserStats))).Methods("GET")
	router.HandleFunc("/api/device/bind_token", withRecovery(middleware.AuthRequired(controllers.GenerateBindToken))).Methods("POST")
	router.HandleFunc("/api/device/list", withRecovery(middleware.AuthRequired(controllers.GetUserDevices))).Methods("GET")
	router.HandleFunc("/api/device/unbind", withRecovery(middleware.AuthRequired(controllers.UnbindDevice))).Methods("POST")
	router.HandleFunc("/api/device/delete", withRecovery(middleware.AuthRequired(controllers.DeleteDevice))).Methods("POST")
	router.HandleFunc("/api/device/rename", withRecovery(middleware.AuthRequired(controllers.RenameDevice))).Methods("POST")
	router.HandleFunc("/api/device/lock", withRecovery(middleware.AuthRequired(controllers.LockDevice))).Methods("POST")
	router.HandleFunc("/api/device/group", withRecovery(middleware.AuthRequired(controllers.SetDeviceGroup))).Methods("POST")
	router.HandleFunc("/api/device/transfer", withRecovery(middleware.AuthRequired(controllers.TransferDevice))).Methods("POST")

	// Subscription API
	router.HandleFunc("/api/subscriptions", withRecovery(middleware.AuthRequired(controllers.CreateSubscription))).Methods("POST")
	router.HandleFunc("/api/subscriptions", withRecovery(middleware.AuthRequired(controllers.GetSubscriptions))).Methods("GET")
	router.HandleFunc("/api/subscriptions/{id}/fetch", withRecovery(middleware.AuthRequired(controllers.FetchVideos(hub)))).Methods("POST")
	router.HandleFunc("/api/subscriptions/{id}/videos", withRecovery(middleware.AuthRequired(controllers.GetSubscriptionVideos))).Methods("GET")
	router.HandleFunc("/api/subscriptions/{id}", withRecovery(middleware.AuthRequired(controllers.DeleteSubscription))).Methods("DELETE")

	// Admin API
	router.HandleFunc("/api/admin/users", withRecovery(middleware.AuthRequired(middleware.AdminRequired(controllers.GetUserList)))).Methods("GET")
	router.HandleFunc("/api/admin/stats", withRecovery(middleware.AuthRequired(middleware.AdminRequired(controllers.GetStats)))).Methods("GET")
	router.HandleFunc("/api/admin/user/credits", withRecovery(middleware.AuthRequired(middleware.AdminRequired(controllers.UpdateUserCredits)))).Methods("POST")
	router.HandleFunc("/api/admin/user/role", withRecovery(middleware.AuthRequired(middleware.AdminRequired(controllers.UpdateUserRole)))).Methods("POST")
	router.HandleFunc("/api/admin/user/{id}", withRecovery(middleware.AuthRequired(middleware.AdminRequired(controllers.DeleteUser)))).Methods("DELETE")
	router.HandleFunc("/api/admin/devices", withRecovery(middleware.AuthRequired(middleware.AdminRequired(controllers.GetAllDevices)))).Methods("GET")
	router.HandleFunc("/api/admin/device/unbind", withRecovery(middleware.AuthRequired(middleware.AdminRequired(controllers.AdminUnbindDevice)))).Methods("POST")
	router.HandleFunc("/api/admin/device/{id}", withRecovery(middleware.AuthRequired(middleware.AdminRequired(controllers.AdminDeleteDevice)))).Methods("DELETE")
	router.HandleFunc("/api/admin/tasks", withRecovery(middleware.AuthRequired(middleware.AdminRequired(controllers.GetAllTasks)))).Methods("GET")
	router.HandleFunc("/api/admin/task/{id}", withRecovery(middleware.AuthRequired(middleware.AdminRequired(controllers.AdminDeleteTask)))).Methods("DELETE")
	router.HandleFunc("/api/admin/subscriptions", withRecovery(middleware.AuthRequired(middleware.AdminRequired(controllers.GetAllSubscriptions)))).Methods("GET")
	router.HandleFunc("/api/admin/subscription/{id}", withRecovery(middleware.AuthRequired(middleware.AdminRequired(controllers.AdminDeleteSubscription)))).Methods("DELETE")

	router.HandleFunc("/api/clients", withRecovery(middleware.AuthRequired(controllers.GetNodes))).Methods("GET")

	router.HandleFunc("/api/tasks", withRecovery(middleware.AuthRequired(controllers.GetTasks))).Methods("GET")
	router.HandleFunc("/api/tasks/detail", withRecovery(middleware.AuthRequired(controllers.GetTaskDetail))).Methods("GET")
	router.HandleFunc("/api/remoteCall", withRecovery(middleware.AuthRequired(controllers.RemoteCall(hub)))).Methods("POST")
	router.HandleFunc("/api/call", withRecovery(middleware.AuthRequired(controllers.RemoteCall(hub)))).Methods("POST")

	// Video Play
	router.HandleFunc("/api/video/play", withRecovery(controllers.PlayVideo)).Methods("GET")

	// Metrics API
	router.HandleFunc("/api/metrics/summary", withRecovery(middleware.AuthRequired(controllers.GetMetricsSummary))).Methods("GET")
	router.HandleFunc("/api/metrics/timeseries", withRecovery(middleware.AuthRequired(controllers.GetTimeSeriesData))).Methods("GET")

	// WebSocket Stats API
	router.HandleFunc("/api/ws/stats", withRecovery(middleware.AuthRequired(controllers.GetWSStats(hub)))).Methods("GET")

	// 静态文件服务 - Vue SPA 支持
	fs := http.FileServer(http.Dir("frontend/dist"))
	router.PathPrefix("/").HandlerFunc(withRecovery(func(w http.ResponseWriter, r *http.Request) {
		// 如果是 API 调用或 WebSocket，不处理
		if strings.HasPrefix(r.URL.Path, "/api/") || strings.HasPrefix(r.URL.Path, "/ws/") {
			http.NotFound(w, r)
			return
		}

		path := r.URL.Path
		// 检查文件是否存在于 dist 目录
		if _, err := os.Stat("frontend/dist" + path); os.IsNotExist(err) {
			// 文件不存在，返回 index.html (SPA History Mode)
			http.ServeFile(w, r, "frontend/dist/index.html")
			return
		}

		// 文件存在，直接服务
		fs.ServeHTTP(w, r)
	}))

	log.Println("Hub Server started on :8080")
	log.Fatal(http.ListenAndServe(":8080", router))
}
