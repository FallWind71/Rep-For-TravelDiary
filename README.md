# MyTravelDiary 旅行日记项目
##操作说明

### cd
change directory.

cd命令用来进入某个目录。例如：
```bash

root@hcss-ecs-50c7:~# cd SJ/
root@hcss-ecs-50c7:~/SJ# 

```
#### 路径和补全
'.'表示当前目录
’..'表示上一个目录
‘～’表示home目录
直接使用cd会返回home目录
按tab键可以进行补全。按两下tab可以显示候选名字。

### ls
list
列出当前目录的所有文件。
list -l列出更加详细的信息。
```bash
root@hcss-ecs-50c7:~/SJ# ls
sleepy
root@hcss-ecs-50c7:~/SJ# ls -l
total 4
drwxr-xr-x 9 root root 4096 Jul 20 00:20 sleepy
```

### mkdir
make directory
创建目录。
```bash
root@hcss-ecs-50c7:~/SJ# ls -l
total 4
drwxr-xr-x 9 root root 4096 Jul 20 00:20 sleepy
root@hcss-ecs-50c7:~/SJ# mkdir test
root@hcss-ecs-50c7:~/SJ# ls -l
total 8
drwxr-xr-x 9 root root 4096 Jul 20 00:20 sleepy
drwxr-xr-x 2 root root 4096 Aug 26 19:07 test
```
### tree
树形结构图
列出当前目录的树形结构图
```bash
root@hcss-ecs-50c7:~/SJ# tree
.
├── sleepy
│   ├── data.json
│   ├── data.py
│   ├── _example
│   │   └── cmd_console.py
│   ├── example.jsonc
│   ├── img
│   │   ├── Cat.jpg
│   │   └── CQSS.jpg
│   ├── img.py
│   ├── install_lib.bat
│   ├── install_lib.sh
│   ├── jsonc_parser
│   │   ├── errors.py
│   │   ├── __init__.py
│   │   ├── jsonc_parser
│   │   │   └── README.md
│   │   ├── LICENSE.txt
│   │   ├── MANIFEST.in
│   │   ├── parser.py
│   │   ├── __pycache__
│   │   │   ├── errors.cpython-310.pyc
│   │   │   ├── __init__.cpython-310.pyc
│   │   │   └── parser.cpython-310.pyc
│   │   ├── README.md
│   │   └── setup.py
│   ├── __pycache__
│   │   ├── data.cpython-310.pyc
│   │   └── utils.cpython-310.pyc
│   ├── README.md
│   ├── requirements.txt
│   ├── server.py
│   ├── start.py
│   ├── static
│   │   └── favicon.ico
│   ├── templates
│   │   ├── index.html
│   │   └── style.css
│   ├── utils.py
│   └── 前台应用状态.macro
└── test

11 directories, 31 files
root@hcss-ecs-50c7:~/SJ# 
```

### rm
remove
删除命令。删除文件夹需要加上‘-r'表示递归删除
```bash
root@hcss-ecs-50c7:~/SJ/test# tree
.
├── test-1
└── test-folder
    └── 666

2 directories, 2 files
root@hcss-ecs-50c7:~/SJ/test# rm test-1
root@hcss-ecs-50c7:~/SJ/test# tree
.
└── test-folder
    └── 666

2 directories, 1 file
root@hcss-ecs-50c7:~/SJ/test# rm test-folder/
rm: cannot remove 'test-folder/': Is a directory
root@hcss-ecs-50c7:~/SJ/test# rm test-folder/ -r
root@hcss-ecs-50c7:~/SJ/test# tree
.

0 directories, 0 files
```
### vim
vim是命令行下常见的文本编辑器，用法是”vim xxx"
```bash
vim test_file.md
```
#### 简略使用方法
```bash
vimtutor
```
该命令可以调出vim的使用教程。

vim文件后，按“i"进入编辑模式，按”Esc“退出编辑模式（进入正常模式）
正常模式下，输入”：“进入命令模式，输入”w“并回车保存文件，输入”q“并回车来退出，输入”wq“并回车从而保存并退出
详细教程参考vimtutor

## 📁 项目结构

```
TD/
├── MyTravelDiary/          # 主要页面目录
│   ├── homepage.html            # 主页（地图界面）
│   ├── zjj.html                # 张家界页面
│   ├── xjp.html                # 新加坡页面
│   ├── szc.html                # 深圳页面
│   ├── nj.html                 # 南京页面
│   ├── gz.html                 # 广州页面
│   ├── mlxy.html               # 马来西亚页面
│   ├── nc.html                 # 南昌页面
│   ├── jj.html                 # 九江页面
│   ├── sz.html                 # 深圳页面
│   ├── server.js               # 后端服务器
│   ├── comments.json           # 评论数据存储
│   ├── package.json            # 项目依赖
│   └── images/                 # 图片资源目录
└── README.md                   # 项目说明文档
```

## 🔍 评论区实现情况分析

### ✅ 真正实现评论区的页面（连接到后端服务器）

以下页面实现了**真正的评论区**，数据会保存到服务器：

1. **zjj.html** (张家界) - 连接到 `http://1.95.203.92:9099`
2. **xjp.html** (新加坡) - 连接到 `http://1.95.203.92:9099`
3. **szc.html** (深圳) - 连接到 `http://1.95.203.92:9099`
4. **nj.html** (南京) - 连接到 `http://1.95.203.92:9099`
5. **gz.html** (广州) - 连接到 `http://1.95.203.92:9099`
6. **mlxy.html** (马来西亚) - 连接到 `http://1.95.203.92:9099`

**特点：**
- 评论数据通过API发送到远程服务器
- 数据持久化存储
- 支持实时加载和提交评论
- 评论包含昵称、内容和时间戳

### ❌ 表面功夫的页面（仅本地存储）

以下页面只有**表面评论区**，数据只存在浏览器本地：

1. **sz.html** (深圳) - 使用 `localStorage` 本地存储
2. **nc.html** (南昌) - 使用 `localStorage` 本地存储  
3. **jj.html** (九江) - 使用 `localStorage` 本地存储

**特点：**
- 评论数据存储在浏览器 `localStorage` 中
- 数据不会同步到服务器
- 清除浏览器数据后评论会丢失
- 不同设备间无法共享评论

## 🚀 如何添加真正的评论区

### 方法1：连接到现有服务器（推荐）

#### 步骤1：修改HTML页面
在需要添加评论区的页面中，找到评论区相关的JavaScript代码，将本地存储改为API调用：

```javascript
// 替换原有的本地存储代码
var cityAbbr = '你的城市缩写'; // 例如：'beijing', 'shanghai'

async function loadMessageList() {
    try {
        const response = await fetch(`http://1.95.203.92:9099/comments/${cityAbbr}`);
        if (!response.ok) throw new Error(`HTTP error! status: ${response.status}`);
        const list = await response.json();
        const messageList = document.getElementById('message-list');
        let html = '';
        if (list.length === 0) {
            html = "<div style='color:#aaa;'>暂无留言</div>";
        } else {
            html = list
                .filter(msg => msg.nick && msg.nick !== '匿名')
                .map((msg, idx) => {
                    const date = new Date(msg.date).toLocaleString('zh-CN', {
                        year: 'numeric',
                        month: '2-digit',
                        day: '2-digit',
                        hour: '2-digit',
                        minute: '2-digit'
                    });
                    return `<div style='margin-bottom:10px;padding:8px 12px;background:#fffbe6;border-radius:6px;border:1px solid #ffe082;word-break:break-all;'>
                                <span style='color:#2196f3;font-weight:bold;margin-right:8px;'>${msg.nick}:</span>
                                <span style='color:#333;'>${msg.text}</span>
                                <span style='float:right;color:#999;font-size:0.95em;'>${date}</span>
                            </div>`;
                })
                .join('');
        }
        messageList.innerHTML = html;
    } catch (err) {
        console.error('加载评论失败:', err);
        document.getElementById('message-list').innerHTML = "<div style='color:#aaa;'>加载评论失败，请稍后重试</div>";
    }
}

async function addMessage() {
    const nickname = document.getElementById('nickname-input').value.trim();
    const text = document.getElementById('message-textarea').value.trim();
    if (!nickname) {
        alert('请填写昵称');
        document.getElementById('nickname-input').focus();
        return;
    }
    if (!text) {
        alert('请输入留言内容');
        return;
    }
    try {
        const response = await fetch(`http://1.95.203.92:9099/comments/${cityAbbr}`, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ nick: nickname, text })
        });
        if (response.ok) {
            document.getElementById('nickname-input').value = '';
            document.getElementById('message-textarea').value = '';
            loadMessageList();
        } else {
            alert('提交评论失败：' + response.statusText);
        }
    } catch (err) {
        console.error('提交评论失败:', err);
        alert('提交评论失败，请检查网络连接');
    }
}
```

#### 步骤2：确保HTML结构完整
确保页面包含以下HTML结构：

```html
<div class="tab-content message-content" style="display:none;">
    <div style='margin-bottom:12px;display:flex;gap:8px;align-items:center;'>
        <span>欢迎留言!!!</span><span style='font-size:40px;'>&#9997;</span><span style='font-size:40px;'>&#9997;</span>
    </div>
    <input id="nickname-input" type="text" placeholder="昵称（必填）" style="width:120px;padding:4px 8px;border-radius:6px;border:1px solid #ffd600;font-size:1em;" maxlength="12">
    <br><br>
    <textarea style="width:100%;height:60px;resize:vertical;border-radius:8px;border:1px solid #ffd600;padding:8px;font-size:1em;box-sizing:border-box;" placeholder="想说些什么呢🤔"></textarea>
    <br>
    <span style='font-size:20px; transform: scaleX(-1); display: inline-block;'>&#128149;</span><span style='font-size:20px;'>&#128147;</span><button id="add-message-btn" style='background:#ffd600;color:#fff;border:none;border-radius:6px;padding:6px 18px;font-size:1em;cursor:pointer;'>添加</button><span style='font-size:20px;'>&#128147;</span><span style='font-size:20px;'>&#128149;</span>
    <div id="message-list" style="margin-top:18px;"></div>
</div>
```

### 方法2：搭建本地服务器

#### 步骤1：安装依赖
```bash
cd MyTravelDiary
npm install
```

#### 步骤2：启动服务器
```bash
npm start
```

服务器将在 `http://localhost:3000` 启动

#### 步骤3：修改页面连接地址
将页面中的API地址改为本地地址：

```javascript
// 将 http://1.95.203.92:9099 改为 http://localhost:3000
const response = await fetch(`http://localhost:3000/comments/${cityAbbr}`);
```

## 📝 城市缩写对照表

| 城市 | 缩写 | 状态 |
|------|------|------|
| 张家界 | zjj | ✅ 真评论区 |
| 新加坡 | xjp | ✅ 真评论区 |
| 深圳 | szc | ✅ 真评论区 |
| 南京 | nj | ✅ 真评论区 |
| 广州 | gz | ✅ 真评论区 |
| 马来西亚 | mlxy | ✅ 真评论区 |
| 深圳 | sz | ❌ 假评论区 |
| 南昌 | nc | ❌ 假评论区 |
| 九江 | jj | ❌ 假评论区 |

## 🔧 技术细节

### 后端API接口

- **GET** `/comments/:city` - 获取指定城市的评论列表
- **POST** `/comments/:city` - 添加新评论到指定城市

### 数据格式

```json
{
  "id": 1,
  "nick": "用户昵称",
  "text": "评论内容",
  "date": "2024-01-01T12:00:00.000Z"
}
```

### 存储方式

- **真评论区**: 数据存储在服务器 `comments.json` 文件中
- **假评论区**: 数据存储在浏览器 `localStorage` 中

## 🚨 注意事项

1. **网络连接**: 真评论区需要网络连接才能正常工作
2. **服务器状态**: 如果远程服务器不可用，真评论区将无法加载
3. **数据安全**: 评论数据存储在远程服务器，请确保服务器安全
4. **浏览器兼容**: 确保浏览器支持 `fetch` API 和 `async/await`

## 📞 技术支持

如果遇到问题，请检查：
1. 网络连接是否正常
2. 服务器是否运行
3. 浏览器控制台是否有错误信息
4. 城市缩写是否正确

---

**最后更新**: 2024年12月
**项目状态**: 部分页面已实现真评论区，部分页面仍为假评论区
