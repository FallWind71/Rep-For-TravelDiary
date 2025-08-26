// MyTravelDiary/server.js
const express = require('express');
const fs = require('fs').promises;
const path = require('path');
const app = express();
const port = 3000;

// 中间件解析 JSON 请求体并提供静态文件
app.use(express.json());
app.use(express.static(path.join(__dirname, '.')));

// 评论存储文件路径
const commentsFile = path.join(__dirname, 'comments.json');

// 初始化评论文件
async function initializeCommentsFile() {
  try {
    await fs.access(commentsFile);
  } catch {
    await fs.writeFile(commentsFile, JSON.stringify({}));
  }
}
initializeCommentsFile();

// GET 端点：获取指定城市的评论
app.get('/comments/:city', async (req, res) => {
  const city = req.params.city;
  try {
    const data = await fs.readFile(commentsFile, 'utf8');
    const comments = JSON.parse(data);
    res.json(comments[city] || []);
  } catch (err) {
    res.status(500).json({ error: '无法读取评论' });
  }
});

// POST 端点：添加新评论
app.post('/comments/:city', async (req, res) => {
  const city = req.params.city;
  const { nick, text } = req.body;
  if (!nick || !text || typeof nick !== 'string' || typeof text !== 'string') {
    return res.status(400).json({ error: '昵称和评论内容不能为空' });
  }

  try {
    const data = await fs.readFile(commentsFile, 'utf8');
    const comments = JSON.parse(data);
    if (!comments[city]) comments[city] = [];
    const newComment = {
      id: comments[city].length ? comments[city][comments[city].length - 1].id + 1 : 1,
      nick: nick,
      text: text,
      date: new Date().toISOString()
    };
    comments[city].push(newComment);
    await fs.writeFile(commentsFile, JSON.stringify(comments, null, 2));
    res.json(newComment);
  } catch (err) {
    res.status(500).json({ error: '无法保存评论' });
  }
});

// 启动服务器
app.listen(port, () => {
  console.log(`服务器运行在 http://localhost:${port}`);
});
