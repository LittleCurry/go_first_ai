// DOM元素
const chatArea = document.getElementById('chatArea');
const messageInput = document.getElementById('messageInput');
const sendBtn = document.getElementById('sendBtn');
const clearBtn = document.getElementById('clearBtn');

// 会话ID
let sessionId = localStorage.getItem('chat_session_id') || generateSessionId();
if (!localStorage.getItem('chat_session_id')) {
    localStorage.setItem('chat_session_id', sessionId);
}

let isWaiting = false;
let currentFullContent = '';
let currentSources = [];

// 生成会话ID
function generateSessionId() {
    return 'session_' + Date.now() + '_' + Math.random().toString(36).substr(2, 9);
}

// 添加消息
function addMessage(role, content, sources = []) {
    const messageDiv = document.createElement('div');
    messageDiv.className = `message ${role}`;

    const avatar = document.createElement('div');
    avatar.className = 'avatar';
    avatar.textContent = role === 'user' ? '👤' : '🤖';

    const bubble = document.createElement('div');
    bubble.className = 'bubble';
    bubble.innerHTML = content.replace(/\n/g, '<br>');

    if (sources && sources.length > 0) {
        const sourcesDiv = document.createElement('div');
        sourcesDiv.className = 'sources';
        sources.forEach(src => {
            const tag = document.createElement('span');
            tag.className = 'tag';
            tag.textContent = `📚 ${src}`;
            sourcesDiv.appendChild(tag);
        });
        bubble.appendChild(sourcesDiv);
    }

    messageDiv.appendChild(role === 'user' ? bubble : avatar);
    messageDiv.appendChild(role === 'user' ? avatar : bubble);

    const welcome = chatArea.querySelector('.welcome-message');
    if (welcome) welcome.remove();

    chatArea.appendChild(messageDiv);
    scrollToBottom();
    return bubble;
}

// 创建流式消息
function createStreamMessage() {
    const messageDiv = document.createElement('div');
    messageDiv.className = 'message bot';
    messageDiv.id = 'stream-message';

    const avatar = document.createElement('div');
    avatar.className = 'avatar';
    avatar.textContent = '🤖';

    const bubble = document.createElement('div');
    bubble.className = 'bubble';
    bubble.innerHTML = '<span class="typing-indicator"><span></span><span></span><span></span></span>';

    messageDiv.appendChild(avatar);
    messageDiv.appendChild(bubble);

    const oldStream = document.getElementById('stream-message');
    if (oldStream) oldStream.remove();

    chatArea.appendChild(messageDiv);
    scrollToBottom();
    return { messageDiv, bubble };
}

// 更新流式消息
function updateStreamMessage(content) {
    const streamMsg = document.getElementById('stream-message');
    if (!streamMsg) return;
    const bubble = streamMsg.querySelector('.bubble');
    if (bubble) {
        bubble.innerHTML = content.replace(/\n/g, '<br>');
        scrollToBottom();
    }
}

// 完成流式消息
function finishStreamMessage(content, sources = []) {
    const streamMsg = document.getElementById('stream-message');
    if (!streamMsg) {
        addMessage('bot', content, sources);
        return;
    }

    const bubble = streamMsg.querySelector('.bubble');
    if (bubble) {
        bubble.innerHTML = content.replace(/\n/g, '<br>');
        if (sources && sources.length > 0) {
            const sourcesDiv = document.createElement('div');
            sourcesDiv.className = 'sources';
            sources.forEach(src => {
                const tag = document.createElement('span');
                tag.className = 'tag';
                tag.textContent = `📚 ${src}`;
                sourcesDiv.appendChild(tag);
            });
            bubble.appendChild(sourcesDiv);
        }
    }
    streamMsg.id = '';
    scrollToBottom();
}

function scrollToBottom() {
    chatArea.scrollTop = chatArea.scrollHeight;
}

// 发送消息（使用Fetch + ReadableStream）
async function sendMessage() {
    const message = messageInput.value.trim();
    if (!message || isWaiting) return;

    messageInput.value = '';
    addMessage('user', message);
    createStreamMessage();

    isWaiting = true;
    sendBtn.disabled = true;
    messageInput.disabled = true;

    currentFullContent = '';
    currentSources = [];

    try {
        const url = `/api/chat/stream?session_id=${encodeURIComponent(sessionId)}&user_id=web_user`;

        const response = await fetch(url, {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json',
            },
            body: JSON.stringify({ message: message }),
        });

        if (!response.ok) {
            throw new Error(`HTTP ${response.status}: ${response.statusText}`);
        }

        const reader = response.body.getReader();
        const decoder = new TextDecoder();
        let buffer = '';

        while (true) {
            const { done, value } = await reader.read();
            if (done) break;

            buffer += decoder.decode(value, { stream: true });
            const lines = buffer.split('\n');
            buffer = lines.pop() || '';

            for (const line of lines) {
                if (line.startsWith('data: ')) {
                    try {
                        const jsonStr = line.slice(6);
                        const data = JSON.parse(jsonStr);

                        if (data.type === 'chunk') {
                            currentFullContent += data.content;
                            updateStreamMessage(currentFullContent);
                        } else if (data.type === 'done') {
                            currentSources = data.sources || [];
                            finishStreamMessage(currentFullContent, currentSources);
                            resetSendState();
                        } else if (data.type === 'error') {
                            finishStreamMessage('抱歉，服务暂时不可用，请稍后重试。');
                            resetSendState();
                        }
                    } catch (e) {
                        console.error('解析SSE数据失败:', e, line);
                    }
                }
            }
        }

        // 如果循环结束但没有收到done消息
        if (isWaiting) {
            if (currentFullContent) {
                finishStreamMessage(currentFullContent + '\n\n（连接已断开）');
            } else {
                finishStreamMessage('抱歉，未收到有效响应，请重试。');
            }
            resetSendState();
        }

    } catch (error) {
        console.error('请求失败:', error);
        finishStreamMessage('抱歉，网络连接失败，请检查网络后重试。');
        resetSendState();
    }
}

function resetSendState() {
    isWaiting = false;
    sendBtn.disabled = false;
    messageInput.disabled = false;
    messageInput.focus();
}

// 清空聊天
function clearChat() {
    chatArea.innerHTML = `
        <div class="welcome-message">
            <div class="avatar">🤖</div>
            <div class="bubble bot">您好！我是AI智能客服助手，请问有什么可以帮您？</div>
        </div>
    `;
    sessionId = generateSessionId();
    localStorage.setItem('chat_session_id', sessionId);
    const streamMsg = document.getElementById('stream-message');
    if (streamMsg) streamMsg.remove();
    resetSendState();
}

// 回车发送
messageInput.addEventListener('keydown', function(e) {
    if (e.key === 'Enter' && !e.shiftKey) {
        e.preventDefault();
        sendMessage();
    }
});

sendBtn.addEventListener('click', sendMessage);
clearBtn.addEventListener('click', clearChat);
messageInput.focus();

console.log('🤖 AI客服已启动，会话ID:', sessionId);