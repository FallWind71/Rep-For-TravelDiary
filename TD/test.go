package main

import (
    "encoding/json"
    "fmt"
    "io"
    "log"
    "net"
    "net/http"
    "os"
    "path/filepath"
    "strings"
    "sync"
    "time"
)

// AccessRecord 结构
type AccessRecord struct {
    IP            string    `json:"ip"`
    UserAgent     string    `json:"user_agent"`
    FirstVisit    time.Time `json:"first_visit"`
    LastVisit     time.Time `json:"last_visit"`
    VisitCount    int       `json:"visit_count"`
    Country       string    `json:"country"`
    Region        string    `json:"region"`
    City          string    `json:"city"`
    ISP           string    `json:"isp"`
    PagesVisited  []string  `json:"pages_visited"`
    Blocked       bool      `json:"blocked"`
    BlockReason   string    `json:"block_reason,omitempty"`
}

// IPGeolocation 结构
type IPGeolocation struct {
    IP          string `json:"ip"`
    Country     string `json:"country"`
    CountryCode string `json:"countryCode"`
    Region      string `json:"region"`
    RegionName  string `json:"regionName"`
    City        string `json:"city"`
    ZIP         string `json:"zip"`
    Lat         float64 `json:"lat"`
    Lon         float64 `json:"lon"`
    Timezone    string `json:"timezone"`
    ISP         string `json:"isp"`
    Org         string `json:"org"`
    AS          string `json:"as"`
    Query       string `json:"query"`
    Status      string `json:"status"`
}

// Comment 结构
type Comment struct {
    ID    int       `json:"id"`
    Nick  string    `json:"nick"`
    Text  string    `json:"text"`
    Date  time.Time `json:"date"`
}

// 全局变量
var (
    accessRecords = make(map[string]*AccessRecord)
    recordsMutex  = sync.RWMutex{}
    comments      = make(map[string][]Comment)
    commentsMutex = sync.RWMutex{}
    logFile       *os.File
    blacklistedIPs = []string{}
    rateLimitPerMinute = 60
    requestCounts     = make(map[string][]time.Time)
    requestMutex      = sync.RWMutex{}
)

func main() {
    initLogFile()
    defer logFile.Close()

    loadAccessRecords()
    loadComments()

    go periodicSave()

    staticDir := "./MyTravelDiary"
    if _, err := os.Stat(staticDir); os.IsNotExist(err) {
        log.Fatal("静态文件目录不存在:", staticDir)
    }

    checkCriticalFiles(staticDir)

    fs := http.FileServer(http.Dir(staticDir))

    http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
        clientIP := getRealIP(r)

        if !securityCheck(clientIP, r) {
            logSecurityEvent(clientIP, r, "BLOCKED")
            http.Error(w, "访问被拒绝", http.StatusForbidden)
            return
        }

        if !rateLimitCheck(clientIP) {
            logSecurityEvent(clientIP, r, "RATE_LIMITED")
            http.Error(w, "请求过多", http.StatusTooManyRequests)
            return
        }

        go recordAccess(clientIP, r)

        log.Printf("请求: %s %s 来自 %s [%s]", r.Method, r.URL.Path, clientIP, r.UserAgent())

        if r.URL.Path == "/" {
            log.Printf("重定向到首页: /homepage.html")
            http.Redirect(w, r, "/homepage.html", http.StatusFound)
            return
        }

        filePath := filepath.Join(staticDir, r.URL.Path)

        if _, err := os.Stat(filePath); os.IsNotExist(err) {
            log.Printf("文件不存在: %s (请求路径: %s) - IP: %s", filePath, r.URL.Path, clientIP)
            w.WriteHeader(http.StatusNotFound)
            w.Header().Set("Content-Type", "text/html; charset=utf-8")
            w.Write([]byte(fmt.Sprintf(`
<!DOCTYPE html>
<html lang="zh-CN">
<head>
    <meta charset="UTF-8">
    <title>页面未找到 - MyTravelDiary</title>
    <style>
        body { 
            font-family: Arial, sans-serif; 
            text-align: center; 
            margin-top: 100px; 
            background: #f5f5f5; 
        }
        .error-container {
            background: white;
            padding: 40px;
            border-radius: 8px;
            box-shadow: 0 2px 10px rgba(0,0,0,0.1);
            max-width: 500px;
            margin: 0 auto;
        }
        h1 { color: #e74c3c; }
        p { color: #666; line-height: 1.6; }
        a { 
            background: #3498db; 
            color: white; 
            padding: 10px 20px; 
            text-decoration: none; 
            border-radius: 4px;
            display: inline-block;
            margin-top: 20px;
        }
        a:hover { background: #2980b9; }
    </style>
</head>
<body>
    <div class="error-container">
        <h1>🚀 页面未找到</h1>
        <p>抱歉，您访问的页面 <strong>%s</strong> 不存在。</p>
        <p>可能是页面正在建设中，或者链接有误。</p>
        <a href="/homepage.html">返回首页</a>
    </div>
</body>
</html>`, r.URL.Path)))
            return
        }

        setContentType(w, r.URL.Path)
        setSecurityHeaders(w)

        if strings.HasSuffix(r.URL.Path, ".html") {
            w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
        } else {
            w.Header().Set("Cache-Control", "public, max-age=3600")
        }

        log.Printf("成功服务文件: %s - IP: %s", filePath, clientIP)
        fs.ServeHTTP(w, r)
    })

    http.HandleFunc("/comments/", func(w http.ResponseWriter, r *http.Request) {
        // 添加 CORS 头
        w.Header().Set("Access-Control-Allow-Origin", "http://1.95.203.92:9099")
        w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
        w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

        // 处理 OPTIONS 预检请求
        if r.Method == http.MethodOptions {
            w.WriteHeader(http.StatusOK)
            return
        }

        city := strings.TrimPrefix(r.URL.Path, "/comments/")
        if city == "" {
            http.Error(w, "无效的城市标识", http.StatusBadRequest)
            return
        }

        switch r.Method {
        case http.MethodGet:
            commentsMutex.RLock()
            defer commentsMutex.RUnlock()
            w.Header().Set("Content-Type", "application/json; charset=utf-8")
            json.NewEncoder(w).Encode(comments[city])
        case http.MethodPost:
            var newComment struct {
                Nick string `json:"nick"`
                Text string `json:"text"`
            }
            if err := json.NewDecoder(r.Body).Decode(&newComment); err != nil {
                http.Error(w, "无效的请求体", http.StatusBadRequest)
                return
            }
            if newComment.Nick == "" || newComment.Text == "" {
                http.Error(w, "昵称和内容不能为空", http.StatusBadRequest)
                return
            }
            commentsMutex.Lock()
            defer commentsMutex.Unlock()
            commentList := comments[city]
            newID := 1
            if len(commentList) > 0 {
                newID = commentList[len(commentList)-1].ID + 1
            }
            comment := Comment{
                ID:    newID,
                Nick:  newComment.Nick,
                Text:  newComment.Text,
                Date:  time.Now(),
            }
            comments[city] = append(commentList, comment)
            w.Header().Set("Content-Type", "application/json; charset=utf-8")
            json.NewEncoder(w).Encode(comment)
        default:
            http.Error(w, "不支持的请求方法", http.StatusMethodNotAllowed)
        }
    })

    http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
        w.Header().Set("Content-Type", "application/json")
        w.Write([]byte(`{"status":"ok","service":"MyTravelDiary"}`))
    })

    http.HandleFunc("/admin/stats", func(w http.ResponseWriter, r *http.Request) {
        if r.Header.Get("X-Admin-Token") != "UbuntuMyTravelDiaryXJWcnm114514!!@" {
            http.Error(w, "未授权", http.StatusUnauthorized)
            return
        }
        recordsMutex.RLock()
        defer recordsMutex.RUnlock()
        w.Header().Set("Content-Type", "application/json; charset=utf-8")
        json.NewEncoder(w).Encode(accessRecords)
    })

    http.HandleFunc("/admin/export", func(w http.ResponseWriter, r *http.Request) {
        if r.Header.Get("X-Admin-Token") != "UbuntuMyTravelDiaryXJWcnm114514!!@" {
            http.Error(w, "未授权", http.StatusUnauthorized)
            return
        }
        w.Header().Set("Content-Type", "application/octet-stream")
        w.Header().Set("Content-Disposition", "attachment; filename=access_records.json")
        recordsMutex.RLock()
        data, _ := json.MarshalIndent(accessRecords, "", "  ")
        recordsMutex.RUnlock()
        w.Write(data)
    })

    log.Println("🌍 MyTravelDiary 增强版服务器已启动")
    log.Println("📍 主页访问地址：http://1.95.203.92:9099")
    log.Println("📍 或直接访问：http://1.95.203.92:9099/homepage.html")
    log.Println("🔍 健康检查：http://1.95.203.92:9099/health")
    log.Println("📊 管理统计：http://1.95.203.92:9099/admin/stats (需要认证)")
    log.Println("📁 导出数据：http://1.95.203.92:9099/admin/export (需要认证)")
    log.Println("🔐 安全特性：IP黑名单、速率限制、地理位置记录已启用")
    log.Println("===========================================")

    err := http.ListenAndServe(":9099", nil)
    if err != nil {
        log.Fatal("❌ 服务器启动失败:", err)
    }
}

func loadComments() {
    data, err := os.ReadFile("comments.json")
    if err != nil {
        log.Println("💾 没有找到评论记录文件，将创建新的记录")
        return
    }
    if err := json.Unmarshal(data, &comments); err != nil {
        log.Printf("⚠  加载评论记录失败: %v", err)
        return
    }
    log.Printf("📊 已加载 %d 个城市的评论记录", len(comments))
}

func periodicSave() {
    ticker := time.NewTicker(5 * time.Minute)
    defer ticker.Stop()
    for {
        select {
        case <-ticker.C:
            saveAccessRecords()
            saveComments()
        }
    }
}

func saveComments() {
    commentsMutex.RLock()
    data, err := json.MarshalIndent(comments, "", "  ")
    commentsMutex.RUnlock()
    if err != nil {
        log.Printf("⚠  序列化评论记录失败: %v", err)
        return
    }
    if err := os.WriteFile("comments.json", data, 0644); err != nil {
        log.Printf("⚠  保存评论记录失败: %v", err)
        return
    }
    log.Println("💾 评论记录已保存")
}

func getRealIP(r *http.Request) string {
    if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
        ips := strings.Split(xff, ",")
        if len(ips) > 0 {
            return strings.TrimSpace(ips[0])
        }
    }
    if xri := r.Header.Get("X-Real-IP"); xri != "" {
        return xri
    }
    ip, _, err := net.SplitHostPort(r.RemoteAddr)
    if err != nil {
        return r.RemoteAddr
    }
    return ip
}

func securityCheck(clientIP string, r *http.Request) bool {
    for _, blockedIP := range blacklistedIPs {
        if clientIP == blockedIP {
            return false
        }
    }
    userAgent := strings.ToLower(r.UserAgent())
    suspiciousAgents := []string{"bot", "crawler", "spider", "scraper"}
    for _, suspicious := range suspiciousAgents {
        if strings.Contains(userAgent, suspicious) {
            log.Printf("⚠  可疑访问: IP %s, User-Agent: %s", clientIP, r.UserAgent())
        }
    }
    suspiciousChars := []string{"../", "..\\", "<script", "<?php", "eval("}
    for _, char := range suspiciousChars {
        if strings.Contains(r.URL.Path, char) {
            log.Printf("🚨 安全警告: 检测到可疑路径 %s from IP %s", r.URL.Path, clientIP)
            return false
        }
    }
    return true
}

func rateLimitCheck(clientIP string) bool {
    requestMutex.Lock()
    defer requestMutex.Unlock()
    now := time.Now()
    cutoff := now.Add(-time.Minute)
    if _, exists := requestCounts[clientIP]; !exists {
        requestCounts[clientIP] = []time.Time{}
    }
    var validRequests []time.Time
    for _, reqTime := range requestCounts[clientIP] {
        if reqTime.After(cutoff) {
            validRequests = append(validRequests, reqTime)
        }
    }
    if len(validRequests) >= rateLimitPerMinute {
        return false
    }
    validRequests = append(validRequests, now)
    requestCounts[clientIP] = validRequests
    return true
}

func recordAccess(clientIP string, r *http.Request) {
    recordsMutex.Lock()
    defer recordsMutex.Unlock()
    record, exists := accessRecords[clientIP]
    if !exists {
        geoInfo := getGeoLocation(clientIP)
        record = &AccessRecord{
            IP:           clientIP,
            UserAgent:    r.UserAgent(),
            FirstVisit:   time.Now(),
            LastVisit:    time.Now(),
            VisitCount:   1,
            Country:      geoInfo.Country,
            Region:       geoInfo.RegionName,
            City:         geoInfo.City,
            ISP:          geoInfo.ISP,
            PagesVisited: []string{r.URL.Path},
            Blocked:      false,
        }
        log.Printf("🆕 新访客: IP %s, 地区: %s %s %s, ISP: %s", 
            clientIP, geoInfo.Country, geoInfo.RegionName, geoInfo.City, geoInfo.ISP)
    } else {
        record.LastVisit = time.Now()
        record.VisitCount++
        found := false
        for _, page := range record.PagesVisited {
            if page == r.URL.Path {
                found = true
                break
            }
        }
        if !found {
            record.PagesVisited = append(record.PagesVisited, r.URL.Path)
        }
    }
    accessRecords[clientIP] = record
    logEntry := fmt.Sprintf("[%s] IP: %s | Path: %s | Agent: %s | Visits: %d | Location: %s %s %s\n",
        time.Now().Format("2006-01-02 15:04:05"),
        clientIP,
        r.URL.Path,
        r.UserAgent(),
        record.VisitCount,
        record.Country,
        record.Region,
        record.City,
    )
    logFile.WriteString(logEntry)
    logFile.Sync()
}

func getGeoLocation(ip string) *IPGeolocation {
    if isLocalIP(ip) {
        return &IPGeolocation{
            IP:      ip,
            Country: "Local",
            Region:  "Local",
            City:    "Local",
            ISP:     "Local Network",
        }
    }
    url := fmt.Sprintf("http://ip-api.com/json/%s?fields=status,message,country,countryCode,region,regionName,city,zip,lat,lon,timezone,isp,org,as,query", ip)
    resp, err := http.Get(url)
    if err != nil {
        log.Printf("获取地理位置信息失败: %v", err)
        return &IPGeolocation{IP: ip, Country: "Unknown", Region: "Unknown", City: "Unknown", ISP: "Unknown"}
    }
    defer resp.Body.Close()
    body, err := io.ReadAll(resp.Body)
    if err != nil {
        log.Printf("读取地理位置响应失败: %v", err)
        return &IPGeolocation{IP: ip, Country: "Unknown", Region: "Unknown", City: "Unknown", ISP: "Unknown"}
    }
    var geoInfo IPGeolocation
    if err := json.Unmarshal(body, &geoInfo); err != nil {
        log.Printf("解析地理位置信息失败: %v", err)
        return &IPGeolocation{IP: ip, Country: "Unknown", Region: "Unknown", City: "Unknown", ISP: "Unknown"}
    }
    if geoInfo.Status != "success" {
        return &IPGeolocation{IP: ip, Country: "Unknown", Region: "Unknown", City: "Unknown", ISP: "Unknown"}
    }
    return &geoInfo
}

func isLocalIP(ip string) bool {
    localRanges := []string{"127.", "192.168.", "10.", "172.16.", "172.17.", "172.18.", "172.19.", 
                          "172.20.", "172.21.", "172.22.", "172.23.", "172.24.", "172.25.", 
                          "172.26.", "172.27.", "172.28.", "172.29.", "172.30.", "172.31.", "::1"}
    for _, localRange := range localRanges {
        if strings.HasPrefix(ip, localRange) {
            return true
        }
    }
    return false
}

func logSecurityEvent(clientIP string, r *http.Request, eventType string) {
    log.Printf("🚨 安全事件 [%s]: IP %s | Path: %s | Agent: %s", 
        eventType, clientIP, r.URL.Path, r.UserAgent())
    recordsMutex.Lock()
    if record, exists := accessRecords[clientIP]; exists {
        record.Blocked = true
        record.BlockReason = eventType
    }
    recordsMutex.Unlock()
}

func setSecurityHeaders(w http.ResponseWriter) {
    w.Header().Set("X-Content-Type-Options", "nosniff")
    w.Header().Set("X-Frame-Options", "DENY")
    w.Header().Set("X-XSS-Protection", "1; mode=block")
    w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
}

func initLogFile() {
    var err error
    logFile, err = os.OpenFile("access.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
    if err != nil {
        log.Printf("⚠  无法创建访问日志文件: %v", err)
        logFile, _ = os.OpenFile(os.DevNull, os.O_WRONLY, 0644)
    } else {
        log.Println("📝 访问日志文件已创建: access.log")
    }
    startupLog := fmt.Sprintf("\n=== 服务器启动 [%s] ===\n", time.Now().Format("2006-01-02 15:04:05"))
    logFile.WriteString(startupLog)
    logFile.Sync()
}

func loadAccessRecords() {
    data, err := os.ReadFile("access_records.json")
    if err != nil {
        log.Println("💾 没有找到历史访问记录文件，将创建新的记录")
        return
    }
    if err := json.Unmarshal(data, &accessRecords); err != nil {
        log.Printf("⚠  加载历史记录失败: %v", err)
        return
    }
    log.Printf("📊 已加载 %d 条历史访问记录", len(accessRecords))
}

func saveAccessRecords() {
    recordsMutex.RLock()
    data, err := json.MarshalIndent(accessRecords, "", "  ")
    recordsMutex.RUnlock()
    if err != nil {
        log.Printf("⚠  序列化访问记录失败: %v", err)
        return
    }
    if err := os.WriteFile("access_records.json", data, 0644); err != nil {
        log.Printf("⚠  保存访问记录失败: %v", err)
        return
    }
    log.Println("💾 访问记录已保存")
}

func setContentType(w http.ResponseWriter, path string) {
    switch {
    case strings.HasSuffix(path, ".html"):
        w.Header().Set("Content-Type", "text/html; charset=utf-8")
    case strings.HasSuffix(path, ".css"):
        w.Header().Set("Content-Type", "text/css; charset=utf-8")
    case strings.HasSuffix(path, ".js"):
        w.Header().Set("Content-Type", "application/javascript; charset=utf-8")
    case strings.HasSuffix(path, ".mp3"):
        w.Header().Set("Content-Type", "audio/mpeg")
    case strings.HasSuffix(path, ".jpg"), strings.HasSuffix(path, ".jpeg"):
        w.Header().Set("Content-Type", "image/jpeg")
    case strings.HasSuffix(path, ".png"):
        w.Header().Set("Content-Type", "image/png")
    case strings.HasSuffix(path, ".gif"):
        w.Header().Set("Content-Type", "image/gif")
    case strings.HasSuffix(path, ".svg"):
        w.Header().Set("Content-Type", "image/svg+xml")
    case strings.HasSuffix(path, ".ico"):
        w.Header().Set("Content-Type", "image/x-icon")
    }
}

func checkCriticalFiles(staticDir string) {
    criticalFiles := []string{
        "homepage.html",
        "nj.html",
        "sz.html",
        "jj.html",
        "nc.html",
        "xjp.html",
        "mlxy.html",
        "zjj.html",
        "gz.html",
    }
    log.Println("🔍 检查关键文件...")
    missingFiles := []string{}
    for _, file := range criticalFiles {
        filePath := filepath.Join(staticDir, file)
        if _, err := os.Stat(filePath); os.IsNotExist(err) {
            missingFiles = append(missingFiles, file)
            log.Printf("⚠  警告: 文件不存在 - %s", file)
        } else {
            log.Printf("✅ 文件存在 - %s", file)
        }
    }
    if len(missingFiles) > 0 {
        log.Printf("⚠  发现 %d 个缺失的HTML文件，这些城市的页面将无法正常访问:", len(missingFiles))
        for _, file := range missingFiles {
            log.Printf("   - %s", file)
        }
        log.Println("💡 建议: 请创建缺失的HTML文件或检查文件名是否正确")
    } else {
        log.Println("✅ 所有关键文件检查完成，没有发现缺失文件")
    }
    resourceDirs := []string{"images", "bgm", "imagesxjp", "imgszc"}
    for _, dir := range resourceDirs {
        dirPath := filepath.Join(staticDir, dir)
        if _, err := os.Stat(dirPath); os.IsNotExist(err) {
            log.Printf("⚠  警告: 资源目录不存在 - %s", dir)
        } else {
            log.Printf("✅ 资源目录存在 - %s", dir)
        }
    }
    log.Println("===========================================")
}
