# Homie - 家庭网络导航页

局域网设备扫描 + 服务发现 + 可视化导航

![Homie Screenshot](screenshot.png)

## 功能特性

- 🌐 **设备扫描** - 自动扫描局域网 IP 和常见端口
- 🔍 **服务识别** - 猜测运行的服务并显示
- ⭐ **快捷链接** - 一键访问局域网服务
- ✏️ **可视化编辑** - 通过 Web GUI 编辑/删除/添加链接
- 💾 **持久化存储** - 链接配置保存在本地 JSON 文件
- 🎨 **简洁美观** - 深色主题，适合家庭 NAS/软路由使用

## 技术栈

- **Go** - 后端 API 服务
- **HTML/CSS/JS** - 前端单页应用
- **Bulma** - CSS 框架（轻量美观）
- **NeteaseCloudMusicApi** 风格的服务发现逻辑

## 快速开始

### 二进制安装

```bash
# 下载对应平台的二进制
./homie -port 8080
```

### 源码运行

```bash
go run main.go -port 8080
```

### Docker 运行

```bash
docker run -d \
  --name homie \
  -p 8080:8080 \
  -v ./data:/app/data \
  ghcr.io/openclaw/homie:latest
```

## 使用说明

1. 启动服务后访问 `http://localhost:8080`
2. 点击「🔄 扫描网络」开始发现局域网设备
3. 扫描完成后，已识别的服务会显示为可点击卡片
4. 点击卡片右上角「✏️」可编辑链接名称和地址
5. 点击「➕ 添加链接」可手动添加自定义链接

## 配置

配置文件位于 `data/links.json`，可直接编辑：

```json
{
  "links": [
    {
      "id": "uuid",
      "name": "OpenClaw",
      "url": "http://192.168.1.100:18789",
      "icon": "🔧",
      "category": "system",
      "created_at": "2025-05-25T00:00:00Z"
    }
  ]
}
```

## API 接口

| 方法 | 路径 | 描述 |
|------|------|------|
| GET | `/api/links` | 获取所有链接 |
| POST | `/api/links` | 添加链接 |
| PUT | `/api/links/:id` | 更新链接 |
| DELETE | `/api/links/:id` | 删除链接 |
| POST | `/api/scan` | 触发网络扫描 |
| GET | `/api/scan/status` | 获取扫描状态 |

## 开发

```bash
# 前端热更新（需要air）
air

# 后端测试
go test ./...
```

## License

MIT
