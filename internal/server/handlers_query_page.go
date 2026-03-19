package server

import (
	"html"
	"net/http"
)

// QueryPageHandler 查询页面处理器
type QueryPageHandler struct {
	queryHandler *QueryHandler
}

// NewQueryPageHandler 创建查询页面处理器
func NewQueryPageHandler(queryHandler *QueryHandler) *QueryPageHandler {
	return &QueryPageHandler{
		queryHandler: queryHandler,
	}
}

// HandleQueryPage 处理查询页面请求
func (h *QueryPageHandler) HandleQueryPage(w http.ResponseWriter, r *http.Request) {
	// 设置CORS头
	h.setCORSHeaders(w)

	// 返回查询页面HTML
	html := h.generateQueryPageHTML()
	h.sendHTMLResponse(w, html, http.StatusOK)
}

// generateQueryPageHTML 生成查询页面HTML
func (h *QueryPageHandler) generateQueryPageHTML() string {
	return `
<!DOCTYPE html>
<html lang="zh-CN">
<head>
    <meta charset="UTF-8">
    <title>🤖 NLQ智能查询</title>
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <style>
        * { margin: 0; padding: 0; box-sizing: border-box; }
        body {
            font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, "Helvetica Neue", Arial, sans-serif;
            background: linear-gradient(135deg, #3498db 0%, #2c3e50 100%);
            display: flex;
            justify-content: center;
            align-items: center;
            min-height: 100vh;
            padding: 20px;
        }
        .container {
            background: white;
            border-radius: 12px;
            box-shadow: 0 10px 40px rgba(0,0,0,0.2);
            max-width: 700px;
            width: 100%;
            padding: 40px;
        }
        .header { text-align: center; margin-bottom: 30px; }
        .emoji { font-size: 64px; margin-bottom: 15px; }
        h1 { font-size: 28px; color: #2c3e50; margin-bottom: 10px; }
        .description { color: #7f8c8d; font-size: 16px; line-height: 1.5; }

        /* 输入区域样式 */
        .input-section { margin-bottom: 20px; }
        .form-group { margin-bottom: 20px; }
        label { display: block; color: #34495e; font-size: 14px; font-weight: 500; margin-bottom: 8px; }
        textarea {
            width: 100%;
            padding: 14px;
            border: 2px solid #ecf0f1;
            border-radius: 8px;
            font-size: 16px;
            font-family: inherit;
            resize: vertical;
            min-height: 120px;
            transition: border-color 0.3s;
        }
        textarea:focus { outline: none; border-color: #3498db; }
        textarea:disabled { background: #f5f5f5; cursor: not-allowed; }

        /* 按钮样式 */
        .button-group { display: flex; gap: 10px; }
        button {
            flex: 1;
            padding: 14px 24px;
            border: none;
            border-radius: 8px;
            font-size: 16px;
            font-weight: 600;
            cursor: pointer;
            transition: all 0.3s;
        }
        .btn-query {
            background: #3498db;
            color: white;
        }
        .btn-query:hover:not(:disabled) {
            background: #2980b9;
            transform: translateY(-2px);
            box-shadow: 0 4px 12px rgba(52, 152, 219, 0.3);
        }
        .btn-query:disabled {
            background: #bdc3c7;
            cursor: not-allowed;
        }
        .btn-clear {
            background: #ecf0f1;
            color: #7f8c8d;
        }
        .btn-clear:hover { background: #bdc3c7; }

        /* 状态消息样式 */
        .loading {
            display: none;
            text-align: center;
            color: #7f8c8d;
            padding: 20px;
            font-size: 16px;
        }
        .loading.show { display: block; }
        .error {
            display: none;
            text-align: center;
            color: #e74c3c;
            padding: 15px;
            background: #fadbd8;
            border-radius: 8px;
            margin: 20px 0;
        }
        .error.show { display: block; }

        /* 结果区域样式 */
        .result-section {
            display: none;
            margin-top: 30px;
        }
        .result-section.show { display: block; }

        .result-header {
            background: #f8f9fa;
            border-radius: 8px;
            padding: 20px;
            margin-bottom: 20px;
        }
        .result-header h3 {
            font-size: 18px;
            color: #2c3e50;
            margin-bottom: 15px;
        }
        .result-item {
            margin-bottom: 12px;
        }
        .result-label {
            font-size: 12px;
            color: #95a5a6;
            text-transform: uppercase;
            margin-bottom: 4px;
        }
        .result-value {
            color: #34495e;
            font-size: 14px;
            word-wrap: break-word;
        }

        /* SQL代码块样式 */
        .sql-code {
            background: #2c3e50;
            color: #ecf0f1;
            padding: 15px;
            border-radius: 6px;
            font-family: 'Monaco', 'Menlo', 'Ubuntu Mono', monospace;
            font-size: 13px;
            line-height: 1.6;
            overflow-x: auto;
            white-space: pre-wrap;
            word-wrap: break-word;
        }

        /* 结果表格样式 */
        .table-container {
            overflow-x: auto;
            margin: 20px 0;
        }
        .result-table {
            width: 100%;
            border-collapse: collapse;
            font-size: 14px;
        }
        .result-table th,
        .result-table td {
            padding: 12px;
            text-align: left;
            border-bottom: 1px solid #ecf0f1;
        }
        .result-table th {
            background: #f8f9fa;
            color: #2c3e50;
            font-weight: 600;
            position: sticky;
            top: 0;
        }
        .result-table tr:hover { background: #f8f9fa; }
        .result-table td { color: #34495e; }
        .empty-result {
            text-align: center;
            color: #7f8c8d;
            padding: 20px;
            font-style: italic;
        }

        /* 展开/收起按钮样式 */
        .toggle-rows-btn {
            display: block;
            width: 100%;
            padding: 10px;
            background: #f8f9fa;
            border: 1px solid #ecf0f1;
            border-radius: 6px;
            color: #3498db;
            font-size: 14px;
            font-weight: 500;
            cursor: pointer;
            transition: all 0.2s;
            text-align: center;
        }
        .toggle-rows-btn:hover {
            background: #e9ecef;
            border-color: #3498db;
        }

        /* 统计信息样式 */
        .stats-info {
            display: flex;
            gap: 20px;
            margin: 15px 0;
            font-size: 14px;
            color: #7f8c8d;
        }
        .stat-item {
            display: flex;
            align-items: center;
            gap: 5px;
        }
        .stat-icon { font-size: 16px; }

        /* 反馈按钮样式 */
        .feedback-section {
            margin-top: 20px;
            padding-top: 20px;
            border-top: 1px solid #ecf0f1;
        }
        .feedback-label {
            font-size: 14px;
            color: #7f8c8d;
            text-align: center;
            margin-bottom: 12px;
        }
        .feedback-buttons {
            display: flex;
            gap: 10px;
        }
        .btn-feedback {
            flex: 1;
            padding: 12px 20px;
            border: none;
            border-radius: 8px;
            font-size: 15px;
            font-weight: 600;
            cursor: pointer;
            transition: all 0.3s;
            display: flex;
            align-items: center;
            justify-content: center;
            gap: 8px;
        }
        .btn-feedback.positive {
            background: #27ae60;
            color: white;
        }
        .btn-feedback.positive:hover {
            background: #229954;
            transform: translateY(-2px);
            box-shadow: 0 4px 12px rgba(39, 174, 96, 0.3);
        }
        .btn-feedback.negative {
            background: #e74c3c;
            color: white;
        }
        .btn-feedback.negative:hover {
            background: #c0392b;
            transform: translateY(-2px);
            box-shadow: 0 4px 12px rgba(231, 76, 60, 0.3);
        }

        /* 导出按钮样式 */
        .btn-export {
            background: #9b59b6;
            color: white;
            padding: 10px 20px;
            border: none;
            border-radius: 6px;
            font-size: 14px;
            font-weight: 600;
            cursor: pointer;
            transition: all 0.3s;
            display: inline-flex;
            align-items: center;
            gap: 6px;
        }
        .btn-export:hover {
            background: #8e44ad;
            transform: translateY(-1px);
            box-shadow: 0 3px 10px rgba(155, 89, 182, 0.3);
        }
        .btn-export:disabled {
            background: #bdc3c7;
            cursor: not-allowed;
            transform: none;
        }

        /* 示例问题样式 */
        .examples {
            margin-top: 15px;
            padding: 15px;
            background: #f8f9fa;
            border-radius: 8px;
        }
        .examples-label {
            font-size: 12px;
            color: #95a5a6;
            text-transform: uppercase;
            margin-bottom: 10px;
        }
        .example-item {
            display: inline-block;
            margin: 5px;
            padding: 6px 12px;
            background: white;
            border: 1px solid #bdc3c7;
            border-radius: 16px;
            font-size: 13px;
            color: #34495e;
            cursor: pointer;
            transition: all 0.2s;
        }
        .example-item:hover {
            background: #3498db;
            color: white;
            border-color: #3498db;
            transform: translateY(-1px);
        }
        .examples-loading {
            font-size: 13px;
            color: #7f8c8d;
            font-style: italic;
        }
        .examples-error {
            font-size: 13px;
            color: #e74c3c;
            font-style: italic;
        }

        /* 响应式设计 */
        @media (max-width: 600px) {
            .container { padding: 20px; }
            .button-group, .feedback-buttons { flex-direction: column; }
            h1 { font-size: 24px; }
            .emoji { font-size: 48px; }
        }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <div class="emoji">🤖</div>
            <h1>NLQ智能查询</h1>
            <p class="description">用自然语言查询数据库，AI自动生成SQL并执行</p>
        </div>

        <!-- 输入区域 -->
        <div class="input-section" id="inputSection">
            <form id="queryForm">
                <div class="form-group">
                    <label for="question">请输入您的问题</label>
                    <textarea
                        id="question"
                        name="question"
                        placeholder="例如：查询VIP用户数量&#10;例如：统计最近7天的订单总数&#10;例如：找出销售额最高的10个产品"
                        required
                    ></textarea>
                </div>

                <!-- 示例问题（动态加载） -->
                <div class="examples">
                    <div class="examples-label">💡 试试这些问题：</div>
                    <div id="examplesContainer">
                        <span class="examples-loading">正在加载示例问题...</span>
                    </div>
                </div>

                <div class="button-group" style="margin-top: 20px;">
                    <button type="submit" class="btn-query" id="queryBtn">
                        🔍 查询
                    </button>
                    <button type="button" class="btn-clear" onclick="clearQuery()">
                        清空
                    </button>
                </div>
            </form>
        </div>

        <!-- 加载状态 -->
        <div class="loading" id="loading">
            <div style="font-size: 48px; margin-bottom: 10px;">⏳</div>
            <div>AI正在思考中，请稍候...</div>
        </div>

        <!-- 错误提示 -->
        <div class="error" id="error"></div>

        <!-- 结果区域 -->
        <div class="result-section" id="resultSection">
            <div class="result-header">
                <h3>📊 查询结果</h3>

                <!-- 问题 -->
                <div class="result-item">
                    <div class="result-label">您的问题</div>
                    <div class="result-value" id="resultQuestion"></div>
                </div>

                <!-- SQL -->
                <div class="result-item">
                    <div class="result-label">生成的SQL</div>
                    <div class="sql-code" id="resultSQL"></div>
                </div>

                <!-- 统计信息 -->
                <div class="stats-info">
                    <div class="stat-item">
                        <span class="stat-icon">⏱️</span>
                        <span id="resultDuration"></span>
                    </div>
                    <div class="stat-item">
                        <span class="stat-icon">📈</span>
                        <span id="resultCount"></span>
                    </div>
                </div>
            </div>

            <!-- 结果表格 -->
            <div class="table-container" id="tableContainer">
                <div style="margin-bottom: 10px; display: flex; justify-content: space-between; align-items: center;">
                    <div style="font-size: 14px; color: #7f8c8d;">
                        <span id="resultInfo"></span>
                    </div>
                    <button class="btn-export" id="exportBtn" onclick="exportToCSV()" style="display: none;">
                        📥 导出CSV
                    </button>
                </div>
                <table class="result-table" id="resultTable">
                    <thead id="resultTableHead"></thead>
                    <tbody id="resultTableBody"></tbody>
                </table>
                <button class="toggle-rows-btn" id="toggleRowsBtn" style="display: none;" onclick="toggleRows()">
                    显示更多 ▼
                </button>
            </div>

            <!-- 反馈按钮 -->
            <div class="feedback-section">
                <div class="feedback-label">这个结果符合预期吗？</div>
                <div class="feedback-buttons">
                    <button class="btn-feedback positive" onclick="openFeedback('positive')">
                        👍 符合预期
                    </button>
                    <button class="btn-feedback negative" onclick="openFeedback('negative')">
                        👎 不符合预期
                    </button>
                </div>
            </div>

            <!-- 新查询按钮 -->
            <div style="margin-top: 20px;">
                <button class="btn-query" style="width: 100%;" onclick="newQuery()">
                    🔄 新查询
                </button>
            </div>
        </div>
    </div>

    <script>
        let currentQueryID = null;
        let currentFeedbackURLs = null;
        let currentQueryData = null; // 保存当前查询结果用于导出
        let currentResultData = null; // 保存结果数据用于展开/收起
        let isExpanded = false; // 是否展开状态
        const INITIAL_ROWS = 10; // 初始显示行数
        const MAX_ROWS = 100; // 最大显示行数

        // 页面加载时加载示例问题
        document.addEventListener('DOMContentLoaded', () => {
            loadSuggestions();
        });

        // 加载示例问题
        async function loadSuggestions() {
            const container = document.getElementById('examplesContainer');

            try {
                const response = await fetch('/api/v1/suggestions');
                const data = await response.json();

                if (data.success && data.suggestions && data.suggestions.length > 0) {
                    // 清空容器
                    container.innerHTML = '';

                    // 渲染示例问题
                    data.suggestions.forEach((suggestion, index) => {
                        const span = document.createElement('span');
                        span.className = 'example-item';
                        span.textContent = suggestion;
                        span.onclick = () => setQuestion(suggestion);
                        container.appendChild(span);
                    });
                } else {
                    container.innerHTML = '<span class="examples-error">暂无示例问题</span>';
                }
            } catch (err) {
                console.error('加载示例问题失败:', err);
                container.innerHTML = '<span class="examples-error">加载失败，请刷新重试</span>';
            }
        }

        // 设置示例问题
        function setQuestion(text) {
            const textarea = document.getElementById('question');
            textarea.value = text;
            textarea.focus();
        }

        // 清空查询
        function clearQuery() {
            document.getElementById('question').value = '';
            document.getElementById('question').focus();
        }

        // 新查询
        function newQuery() {
            // 隐藏结果区域
            document.getElementById('resultSection').classList.remove('show');
            // 显示输入区域
            document.getElementById('inputSection').style.display = 'block';
            // 清空输入
            clearQuery();
            // 重置状态
            currentQueryID = null;
            currentFeedbackURLs = null;
        }

        // 显示加载状态
        function showLoading() {
            document.getElementById('loading').classList.add('show');
            document.getElementById('error').classList.remove('show');
            document.getElementById('resultSection').classList.remove('show');
            document.getElementById('queryBtn').disabled = true;
            document.getElementById('question').disabled = true;
        }

        // 隐藏加载状态
        function hideLoading() {
            document.getElementById('loading').classList.remove('show');
            document.getElementById('queryBtn').disabled = false;
            document.getElementById('question').disabled = false;
        }

        // 显示错误
        function showError(message) {
            const errorEl = document.getElementById('error');
            errorEl.textContent = '❌ ' + message;
            errorEl.classList.add('show');
            setTimeout(() => {
                errorEl.classList.remove('show');
            }, 5000);
        }

        // 显示结果
        function showResult(data) {
            // 隐藏加载状态和输入区域
            hideLoading();
            document.getElementById('inputSection').style.display = 'none';

            // 保存查询结果用于导出
            currentQueryData = data;

            // 显示问题
            document.getElementById('resultQuestion').textContent = data.question || '';

            // 显示SQL
            document.getElementById('resultSQL').textContent = data.sql || '';

            // 显示统计信息
            document.getElementById('resultDuration').textContent =
                (data.duration_ms / 1000).toFixed(2) + ' 秒';
            document.getElementById('resultCount').textContent =
                (data.count || 0) + ' 条记录';

            // 显示结果表格
            renderResultTable(data.result || []);

            // 显示导出按钮（如果有结果）
            const exportBtn = document.getElementById('exportBtn');
            if (data.result && data.result.length > 0) {
                exportBtn.style.display = 'inline-flex';
            } else {
                exportBtn.style.display = 'none';
            }

            // 保存QueryID和反馈链接
            currentQueryID = data.query_id;
            currentFeedbackURLs = data.feedback;

            // 显示结果区域
            document.getElementById('resultSection').classList.add('show');
        }

        // 渲染结果表格
        function renderResultTable(results) {
            const thead = document.getElementById('resultTableHead');
            const tbody = document.getElementById('resultTableBody');
            const toggleBtn = document.getElementById('toggleRowsBtn');

            // 保存完整结果数据
            currentResultData = results;
            isExpanded = false; // 重置展开状态

            // 清空表格
            thead.innerHTML = '';
            tbody.innerHTML = '';

            if (!results || results.length === 0) {
                tbody.innerHTML = '<tr><td colspan="100%"><div class="empty-result">暂无数据</div></td></tr>';
                toggleBtn.style.display = 'none';
                return;
            }

            // 获取所有列名
            const columns = Object.keys(results[0]);

            // 创建表头
            const headerRow = document.createElement('tr');
            columns.forEach(col => {
                const th = document.createElement('th');
                th.textContent = col;
                headerRow.appendChild(th);
            });
            thead.appendChild(headerRow);

            // 判断是否需要折叠
            const needsCollapse = results.length > INITIAL_ROWS;
            const displayRows = needsCollapse ? INITIAL_ROWS : Math.min(results.length, MAX_ROWS);

            // 创建数据行
            for (let i = 0; i < displayRows; i++) {
                const row = results[i];
                const tr = document.createElement('tr');
                columns.forEach(col => {
                    const td = document.createElement('td');
                    const value = row[col];
                    // 处理null值
                    td.textContent = (value === null || value === undefined) ? 'NULL' : String(value);
                    tr.appendChild(td);
                });
                tbody.appendChild(tr);
            }

            // 如果数据超过初始行数，显示折叠按钮
            if (needsCollapse) {
                toggleBtn.style.display = 'block';
                if (results.length > MAX_ROWS) {
                    toggleBtn.textContent = '显示更多 ▼ (前' + INITIAL_ROWS + '条，共' + results.length + '条)';
                } else {
                    toggleBtn.textContent = '显示更多 ▼ (前' + INITIAL_ROWS + '条，共' + results.length + '条)';
                }
            } else {
                toggleBtn.style.display = 'none';
            }

            // 更新结果信息
            const resultInfo = document.getElementById('resultInfo');
            if (resultInfo) {
                resultInfo.textContent = '共 ' + results.length + ' 条记录';
            }
        }

        // 切换展开/收起状态
        function toggleRows() {
            if (!currentResultData || currentResultData.length === 0) {
                return;
            }

            const tbody = document.getElementById('resultTableBody');
            const toggleBtn = document.getElementById('toggleRowsBtn');
            const results = currentResultData;
            const columns = Object.keys(results[0]);

            // 清空tbody（保留表头）
            tbody.innerHTML = '';

            if (isExpanded) {
                // 收起：只显示初始行数
                for (let i = 0; i < INITIAL_ROWS; i++) {
                    const row = results[i];
                    const tr = document.createElement('tr');
                    columns.forEach(col => {
                        const td = document.createElement('td');
                        const value = row[col];
                        td.textContent = (value === null || value === undefined) ? 'NULL' : String(value);
                        tr.appendChild(td);
                    });
                    tbody.appendChild(tr);
                }

                toggleBtn.textContent = '显示更多 ▼ (前' + INITIAL_ROWS + '条，共' + results.length + '条)';
                isExpanded = false;
            } else {
                // 展开：显示最大行数
                const maxRows = Math.min(results.length, MAX_ROWS);
                for (let i = 0; i < maxRows; i++) {
                    const row = results[i];
                    const tr = document.createElement('tr');
                    columns.forEach(col => {
                        const td = document.createElement('td');
                        const value = row[col];
                        td.textContent = (value === null || value === undefined) ? 'NULL' : String(value);
                        tr.appendChild(td);
                    });
                    tbody.appendChild(tr);
                }

                // 如果超过最大行数，添加提示行
                if (results.length > MAX_ROWS) {
                    const tr = document.createElement('tr');
                    const td = document.createElement('td');
                    td.colSpan = columns.length;
                    td.innerHTML = '<div class="empty-result">仅显示前' + MAX_ROWS + '条数据（共' + results.length + '条），请使用导出功能获取完整数据</div>';
                    tr.appendChild(td);
                    tbody.appendChild(tr);
                    toggleBtn.textContent = '收起 ▲';
                } else {
                    toggleBtn.textContent = '收起 ▲';
                }
                isExpanded = true;
            }

            // 滚动到表格位置
            document.getElementById('resultTable').scrollIntoView({ behavior: 'smooth', block: 'nearest' });
        }

        // 打开反馈页面
        function openFeedback(type) {
            if (!currentFeedbackURLs) {
                showError('反馈链接不可用');
                return;
            }

            const url = type === 'positive' ?
                currentFeedbackURLs.positive_url :
                currentFeedbackURLs.negative_url;

            if (url) {
                window.open(url, '_blank');
            } else {
                showError('反馈链接不可用');
            }
        }

        // 查询表单提交
        document.getElementById('queryForm').addEventListener('submit', async (e) => {
            e.preventDefault();

            const question = document.getElementById('question').value.trim();
            if (!question) {
                showError('请输入您的问题');
                return;
            }

            showLoading();

            try {
                const response = await fetch('/api/v1/query', {
                    method: 'POST',
                    headers: { 'Content-Type': 'application/json' },
                    body: JSON.stringify({ question })
                });

                const data = await response.json();

                if (data.success) {
                    showResult(data);
                } else {
                    showError(data.error || '查询失败，请稍后重试');
                }
            } catch (err) {
                console.error('查询错误:', err);
                showError('网络错误：' + err.message);
            } finally {
                hideLoading();
            }
        });

        // 支持Ctrl+Enter提交
        document.getElementById('question').addEventListener('keydown', (e) => {
            if ((e.ctrlKey || e.metaKey) && e.key === 'Enter') {
                e.preventDefault();
                document.getElementById('queryForm').dispatchEvent(new Event('submit'));
            }
        });

        // 导出CSV
        function exportToCSV() {
            if (!currentQueryData || !currentQueryData.result || currentQueryData.result.length === 0) {
                showError('没有可导出的数据');
                return;
            }

            try {
                const results = currentQueryData.result;
                const columns = Object.keys(results[0]);

                // 生成CSV内容
                let csv = convertToCSV(results, columns);

                // 添加BOM以支持Excel正确显示中文
                const bom = '\uFEFF';
                const blob = new Blob([bom + csv], { type: 'text/csv;charset=utf-8;' });

                // 创建下载链接
                const link = document.createElement('a');
                const url = URL.createObjectURL(blob);
                const timestamp = new Date().toISOString().replace(/[:.]/g, '-').slice(0, -5);
                const filename = 'query_result_' + timestamp + '.csv';

                link.setAttribute('href', url);
                link.setAttribute('download', filename);
                link.style.visibility = 'hidden';
                document.body.appendChild(link);
                link.click();
                document.body.removeChild(link);

                // 释放URL对象
                setTimeout(() => URL.revokeObjectURL(url), 100);
            } catch (err) {
                console.error('导出CSV失败:', err);
                showError('导出失败: ' + err.message);
            }
        }

        // 转换数据为CSV格式
        function convertToCSV(data, columns) {
            let csv = '';

            // 添加表头
            csv += columns.map(col => escapeCSVField(col)).join(',') + '\n';

            // 添加数据行
            data.forEach(row => {
                const values = columns.map(col => {
                    const value = row[col];
                    return escapeCSVField(formatCSVValue(value));
                });
                csv += values.join(',') + '\n';
            });

            return csv;
        }

        // 格式化值用于CSV
        function formatCSVValue(value) {
            if (value === null || value === undefined) {
                return '';
            }
            if (typeof value === 'object') {
                return JSON.stringify(value);
            }
            return String(value);
        }

        // 转义CSV字段（处理逗号、引号、换行符）
        function escapeCSVField(field) {
            const str = String(field);
            // 如果包含逗号、引号或换行符，需要用引号包裹并转义内部引号
            if (str.includes(',') || str.includes('"') || str.includes('\n') || str.includes('\r')) {
                return '"' + str.replace(/"/g, '""') + '"';
            }
            return str;
        }
    </script>
</body>
</html>`
}

// escapeHTML 转义HTML特殊字符
func (h *QueryPageHandler) escapeHTML(text string) string {
	return html.EscapeString(text)
}

// sendHTMLResponse 发送HTML响应
func (h *QueryPageHandler) sendHTMLResponse(w http.ResponseWriter, htmlStr string, statusCode int) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(statusCode)
	w.Write([]byte(htmlStr))
}

// setCORSHeaders 设置CORS头
func (h *QueryPageHandler) setCORSHeaders(w http.ResponseWriter) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
}
