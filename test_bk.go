package main

import (
    "encoding/json"
    "fmt"
    "io/ioutil"
    "log"
    "net"
    "net/http"
    "os"
    "path/filepath"
    "strings"
    "sync"
    "time"
)

// 访问记录结构
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

// IP地理位置信息
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

// 全局访问记录存储
var (
    accessRecords = make(map[string]*AccessRecord)
    recordsMutex  = sync.RWMutex{}
    logFile      *os.File
    
    // 黑名单IP
    blacklistedIPs = []string{
        // 可以在这里添加需要屏蔽的IP
        // "192.168.1.100",
    }
    
    // 速率限制：每IP每分钟最大请求数
    rateLimitPerMinute = 60
    requestCounts     = make(map[string][]time.Time)
    requestMutex      = sync.RWMutex{}
)

func main() {
    // 初始化日志文件
    initLogFile()
    defer logFile.Close()
    
    // 加载历史访问记录
    loadAccessRecords()
    
    // 定期保存访问记录
    go periodicSave()
    
    // 设置静态文件目录
    staticDir := "./MyTravelDiary"
    
    // 检查静态文件目录是否存在
    if _, err := os.Stat(staticDir); os.IsNotExist(err) {
        log.Fatal("静态文件目录不存在:", staticDir)
    }
    
    // 预检查关键HTML文件是否存在
    checkCriticalFiles(staticDir)
    
    // 自定义文件服务器处理器
    fs := http.FileServer(http.Dir(staticDir))
    
    // 处理所有请求
    http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
        // 获取真实IP
        clientIP := getRealIP(r)
        
        // 安全检查
        if !securityCheck(clientIP, r) {
            logSecurityEvent(clientIP, r, "BLOCKED")
            http.Error(w, "Access Denied", http.StatusForbidden)
            return
        }
        
        // 速率限制检查
        if !rateLimitCheck(clientIP) {
            logSecurityEvent(clientIP, r, "RATE_LIMITED")
            http.Error(w, "Too Many Requests", http.StatusTooManyRequests)
            return
        }
        
        // 记录访问信息（异步处理）
        go recordAccess(clientIP, r)
        
        // 记录请求
        log.Printf("请求: %s %s 来自 %s [%s]", r.Method, r.URL.Path, clientIP, r.UserAgent())
        
        // 如果请求根路径，重定向到homepage.html
        if r.URL.Path == "/" {
            log.Printf("重定向到首页: /homepage.html")
            http.Redirect(w, r, "/homepage.html", http.StatusFound)
            return
        }
        
        // 构建文件路径
        filePath := filepath.Join(staticDir, r.URL.Path)
        
        // 检查文件是否存在
        if _, err := os.Stat(filePath); os.IsNotExist(err) {
            log.Printf("文件不存在: %s (请求路径: %s) - IP: %s", filePath, r.URL.Path, clientIP)
            
            // 返回更友好的404页面
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
        
        // 设置正确的Content-Type
        setContentType(w, r.URL.Path)
        
        // 添加安全头
        setSecurityHeaders(w)
        
        // 添加缓存控制头
        if strings.HasSuffix(r.URL.Path, ".html") {
            w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
        } else {
            w.Header().Set("Cache-Control", "public, max-age=3600")
        }
        
        log.Printf("成功服务文件: %s - IP: %s", filePath, clientIP)
        
        // 服务文件
        fs.ServeHTTP(w, r)
    })
    
    // 健康检查端点
    http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
        w.Header().Set("Content-Type", "application/json")
        w.Write([]byte(`{"status":"ok","service":"MyTravelDiary"}`))
    })
    
    // 管理端点 - 查看访问统计
    http.HandleFunc("/admin/stats", func(w http.ResponseWriter, r *http.Request) {
        // 简单的认证检查（在生产环境中应使用更安全的认证方式）
        if r.Header.Get("X-Admin-Token") != "UbuntuMyTravelDiaryXJWcnm114514!!@" {
            http.Error(w, "Unauthorized", http.StatusUnauthorized)
            return
        }
        
        recordsMutex.RLock()
        defer recordsMutex.RUnlock()
        
        w.Header().Set("Content-Type", "application/json; charset=utf-8")
        json.NewEncoder(w).Encode(accessRecords)
    })
    
    // 导出访问日志端点
    http.HandleFunc("/admin/export", func(w http.ResponseWriter, r *http.Request) {
        if r.Header.Get("X-Admin-Token") != "UbuntuMyTravelDiaryXJWcnm114514!!@" {
            http.Error(w, "Unauthorized", http.StatusUnauthorized)
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
    log.Println("📍 主页访问地址：http://localhost:9099")
    log.Println("📍 或直接访问：http://localhost:9099/homepage.html")
    log.Println("🔍 健康检查：http://localhost:9099/health")
    log.Println("📊 管理统计：http://localhost:9099/admin/stats (需要认证)")
    log.Println("📁 导出数据：http://localhost:9099/admin/export (需要认证)")
    log.Println("🔐 安全特性：IP黑名单、速率限制、地理位置记录已启用")
    log.Println("===========================================")
    
    err := http.ListenAndServe(":9099", nil)
    if err != nil {
        log.Fatal("❌ 服务器启动失败:", err)
    }
}

// 获取真实IP地址
func getRealIP(r *http.Request) string {
    // 检查 X-Forwarded-For 头
    if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
        ips := strings.Split(xff, ",")
        if len(ips) > 0 {
            return strings.TrimSpace(ips[0])
        }
    }
    
    // 检查 X-Real-IP 头
    if xri := r.Header.Get("X-Real-IP"); xri != "" {
        return xri
    }
    
    // 从 RemoteAddr 获取
    ip, _, err := net.SplitHostPort(r.RemoteAddr)
    if err != nil {
        return r.RemoteAddr
    }
    return ip
}

// 安全检查
func securityCheck(clientIP string, r *http.Request) bool {
    // 检查IP黑名单
    for _, blockedIP := range blacklistedIPs {
        if clientIP == blockedIP {
            return false
        }
    }
    
    // 检查可疑的User-Agent
    userAgent := strings.ToLower(r.UserAgent())
    suspiciousAgents := []string{"bot", "crawler", "spider", "scraper"}
    for _, suspicious := range suspiciousAgents {
        if strings.Contains(userAgent, suspicious) {
            // 可以选择阻止或仅记录
            log.Printf("⚠️  可疑访问: IP %s, User-Agent: %s", clientIP, r.UserAgent())
            // return false // 如果要阻止，取消注释这行
        }
    }
    
    // 检查请求路径是否包含可疑字符
    suspiciousChars := []string{"../", "..\\", "<script", "<?php", "eval("}
    for _, char := range suspiciousChars {
        if strings.Contains(r.URL.Path, char) {
            log.Printf("🚨 安全警告: 检测到可疑路径 %s from IP %s", r.URL.Path, clientIP)
            return false
        }
    }
    
    return true
}

// 速率限制检查
func rateLimitCheck(clientIP string) bool {
    requestMutex.Lock()
    defer requestMutex.Unlock()
    
    now := time.Now()
    cutoff := now.Add(-time.Minute)
    
    // 获取该IP的请求历史
    if _, exists := requestCounts[clientIP]; !exists {
        requestCounts[clientIP] = []time.Time{}
    }
    
    // 清理过期的请求记录
    var validRequests []time.Time
    for _, reqTime := range requestCounts[clientIP] {
        if reqTime.After(cutoff) {
            validRequests = append(validRequests, reqTime)
        }
    }
    
    // 检查是否超过限制
    if len(validRequests) >= rateLimitPerMinute {
        return false
    }
    
    // 记录当前请求
    validRequests = append(validRequests, now)
    requestCounts[clientIP] = validRequests
    
    return true
}

// 记录访问信息
func recordAccess(clientIP string, r *http.Request) {
    recordsMutex.Lock()
    defer recordsMutex.Unlock()
    
    // 检查是否已有该IP的记录
    record, exists := accessRecords[clientIP]
    if !exists {
        // 获取地理位置信息
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
        // 更新现有记录
        record.LastVisit = time.Now()
        record.VisitCount++
        
        // 记录访问的页面（避免重复）
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
    
    // 写入访问日志文件
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
    logFile.Sync() // 立即写入磁盘
}

// 获取地理位置信息
func getGeoLocation(ip string) *IPGeolocation {
    // 检查是否为本地IP
    if isLocalIP(ip) {
        return &IPGeolocation{
            IP:      ip,
            Country: "Local",
            Region:  "Local",
            City:    "Local",
            ISP:     "Local Network",
        }
    }
    
    // 使用免费的ip-api.com服务获取地理位置信息
    url := fmt.Sprintf("http://ip-api.com/json/%s?fields=status,message,country,countryCode,region,regionName,city,zip,lat,lon,timezone,isp,org,as,query", ip)
    
    resp, err := http.Get(url)
    if err != nil {
        log.Printf("获取地理位置信息失败: %v", err)
        return &IPGeolocation{IP: ip, Country: "Unknown", Region: "Unknown", City: "Unknown", ISP: "Unknown"}
    }
    defer resp.Body.Close()
    
    body, err := ioutil.ReadAll(resp.Body)
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

// 检查是否为本地IP
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

// 记录安全事件
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

// 设置安全响应头
func setSecurityHeaders(w http.ResponseWriter) {
    w.Header().Set("X-Content-Type-Options", "nosniff")
    w.Header().Set("X-Frame-Options", "DENY")
    w.Header().Set("X-XSS-Protection", "1; mode=block")
    w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
}

// 初始化日志文件
func initLogFile() {
    var err error
    logFile, err = os.OpenFile("access.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
    if err != nil {
        log.Printf("⚠️  无法创建访问日志文件: %v", err)
        // 创建一个虚拟的文件句柄以避免崩溃
        logFile, _ = os.OpenFile(os.DevNull, os.O_WRONLY, 0644)
    } else {
        log.Println("📝 访问日志文件已创建: access.log")
    }
    
    // 写入启动标记
    startupLog := fmt.Sprintf("\n=== 服务器启动 [%s] ===\n", time.Now().Format("2006-01-02 15:04:05"))
    logFile.WriteString(startupLog)
    logFile.Sync()
}

// 加载历史访问记录
func loadAccessRecords() {
    data, err := ioutil.ReadFile("access_records.json")
    if err != nil {
        log.Println("💾 没有找到历史访问记录文件，将创建新的记录")
        return
    }
    
    if err := json.Unmarshal(data, &accessRecords); err != nil {
        log.Printf("⚠️  加载历史记录失败: %v", err)
        return
    }
    
    log.Printf("📊 已加载 %d 条历史访问记录", len(accessRecords))
}

// 定期保存访问记录
func periodicSave() {
    ticker := time.NewTicker(5 * time.Minute) // 每5分钟保存一次
    defer ticker.Stop()
    
    for {
        select {
        case <-ticker.C:
            saveAccessRecords()
        }
    }
}

// 保存访问记录到文件
func saveAccessRecords() {
    recordsMutex.RLock()
    data, err := json.MarshalIndent(accessRecords, "", "  ")
    recordsMutex.RUnlock()
    
    if err != nil {
        log.Printf("⚠️  序列化访问记录失败: %v", err)
        return
    }
    
    if err := ioutil.WriteFile("access_records.json", data, 0644); err != nil {
        log.Printf("⚠️  保存访问记录失败: %v", err)
        return
    }
    
    log.Println("💾 访问记录已保存")
}

// 设置Content-Type
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

// 检查关键文件是否存在
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
        "szc.html",
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
    
    // 检查资源目录
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
