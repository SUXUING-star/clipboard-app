import { useState, useEffect } from 'react';
import { Card, CardContent } from "./components/ui/card";
import { Copy, Trash2, Minus } from 'lucide-react'; // 添加 Minus
import { PreviewModal } from './components/PreviewModal';
import { toast, Toaster } from 'react-hot-toast';

function App() {
	const [clipboardHistory, setClipboardHistory] = useState([]);
	const [selectedType, setSelectedType] = useState('all');
	const [previewItem, setPreviewItem] = useState(null);
	const [autoStart, setAutoStart] = useState(false);

	useEffect(() => {
		// 获取初始历史记录和自启动状态
		window.go.main.App.GetClipboardHistory().then(setClipboardHistory);
		window.go.main.App.CheckAutoStart().then(setAutoStart);

		// 监听更新
		window.runtime.EventsOn("clipboard-update", (history) => {
			setClipboardHistory(history);
			// 判断是新增的内容
			if (history.length > 0 && (!clipboardHistory.length ||
				history[0].timestamp !== clipboardHistory[0]?.timestamp)) {
				const type = history[0].type;
				toast.success(`新的${type === 'text' ? '文本' : '图片'}已添加到历史记录`, {
					duration: 2000,
				});
			}
		});

		return () => {
			window.runtime.EventsOff("clipboard-update");
		};
	}, [clipboardHistory]);

	const handleCopy = async (e, item) => {
		e.stopPropagation();
		try {
				if (item.type === 'text') {
						await window.go.main.App.SetClipboardContent(item.content);
						toast.success('文本已复制到剪贴板');
				} else if (item.type === 'image') {
						const img = new Image();
						img.src = item.content;
						img.onload = () => {
								const canvas = document.createElement('canvas');
								canvas.width = img.width;
								canvas.height = img.height;
								const ctx = canvas.getContext('2d');
								ctx.drawImage(img, 0, 0);
								canvas.toBlob(async (blob) => {
										try {
												await navigator.clipboard.write([
														new ClipboardItem({ 'image/png': blob })
												]);
												toast.success('图片已复制到剪贴板');
										} catch (err) {
												toast.error('复制图片失败');
										}
								}, 'image/png');
						};
				}
		} catch (err) {
				toast.error('复制失败');
		}
	};

	const handlePreview = (item) => {
		setPreviewItem(item);
	};

	const handleClearHistory = async () => {
		try {
			await window.go.main.App.ClearHistory();
			toast.success('历史记录已清空');
		} catch (err) {
			console.error('清空历史失败:', err);
			toast.error('清空历史失败');
		}
	};

	const handleAutoStartChange = async (checked) => {
		try {
			await window.go.main.App.SetAutoStart(checked);
			setAutoStart(checked);
			toast.success(checked ? '已设置开机自启' : '已取消开机自启');
		} catch (err) {
			console.error('设置自启动失败:', err);
			toast.error('设置失败：' + err.message);
			// 恢复选中状态
			setAutoStart(!checked);
		}
	};


	const filteredHistory = clipboardHistory.filter(item =>
		selectedType === 'all' || item.type === selectedType
	);

	return (
		<>
			<div className="flex h-screen bg-gray-100">
				{/* 左侧工具栏 */}
				<div className="w-48 bg-white shadow-md p-4 flex flex-col gap-3">
					<div className="flex justify-between items-center mb-4">
						<h1 className="text-xl font-bold">剪贴板历史</h1>
						<button
							onClick={() => window.go.main.App.WindowHide()}
							className="p-1 hover:bg-gray-100 rounded"
							title="最小化到托盘"
						>
							<Minus size={16} />
						</button>
					</div>

					<button
						onClick={() => setSelectedType('all')}
						className={`w-full px-4 py-2 rounded text-left ${selectedType === 'all'
							? 'bg-blue-500 text-white'
							: 'bg-gray-100 hover:bg-gray-200'
							}`}
					>
						全部
					</button>
					<button
						onClick={() => setSelectedType('text')}
						className={`w-full px-4 py-2 rounded text-left ${selectedType === 'text'
							? 'bg-blue-500 text-white'
							: 'bg-gray-100 hover:bg-gray-200'
							}`}
					>
						文本
					</button>
					<button
						onClick={() => setSelectedType('image')}
						className={`w-full px-4 py-2 rounded text-left ${selectedType === 'image'
							? 'bg-blue-500 text-white'
							: 'bg-gray-100 hover:bg-gray-200'
							}`}
					>
						图片
					</button>
					<div className="mt-auto space-y-2">
						{/* 设置区域 */}
						<div className="mt-auto space-y-2">
							<div className="flex items-center justify-between px-4 py-2">
								<span>开机自启</span>
								<input
									type="checkbox"
									checked={autoStart}
									onChange={(e) => handleAutoStartChange(e.target.checked)}
									className="form-checkbox h-4 w-4 text-blue-500"
								/>
							</div>
							<Toaster
								position="top-center"
								toastOptions={{
									duration: 2000,
									style: {
										background: '#333',
										color: '#fff',
									},
									success: {
										style: {
											background: 'green',
										},
									},
									error: {
										style: {
											background: 'red',
										},
									},
								}}
							/>
							<button
								onClick={handleClearHistory}
								className="w-full px-4 py-2 rounded text-left bg-red-500 text-white hover:bg-red-600 flex items-center gap-2"
							>
								<Trash2 size={16} />
								清空历史
							</button>
						</div>
					</div>
				</div>

				{/* 右侧内容区域 */}
				<div className="flex-1 p-4 overflow-y-auto">
					<div className="space-y-4">
						{filteredHistory.length === 0 ? (
							<div className="text-center text-gray-500 py-8">
								暂无剪贴板历史记录
							</div>
						) : (
							filteredHistory.map((item) => (
								<Card
									key={item.timestamp}
									className="hover:shadow-md transition-all cursor-pointer relative"
									onClick={() => handlePreview(item)}
								>
									<CardContent className="p-4">
										<div className="flex justify-between items-center mb-2">
											<span className="text-sm text-gray-500">
												{new Date(item.timestamp).toLocaleString()}
											</span>
											<div className="flex items-center gap-2">
												<span className="text-xs px-2 py-1 rounded-full bg-gray-100">
													{item.type === 'text' ? '文本' : '图片'}
												</span>
												<button
													onClick={(e) => handleCopy(e, item)}
													className="p-1 hover:bg-gray-100 rounded"
													title="复制内容"
												>
													<Copy size={16} />
												</button>
											</div>
										</div>
										{item.type === 'text' ? (
											<div className="whitespace-pre-wrap break-words line-clamp-3 hover:text-blue-500">
												{item.content}
											</div>
										) : (
											<div className="cursor-zoom-in">
												<img
													src={item.content}
													alt="剪贴板图片"
													className="max-h-48 object-contain mx-auto hover:opacity-90"
												/>
											</div>
										)}
									</CardContent>
								</Card>
							))
						)}
					</div>
				</div>
			</div>

			{/* 预览模态框 */}
			<PreviewModal
				isOpen={!!previewItem}
				onClose={() => setPreviewItem(null)}
				content={previewItem?.content}
				type={previewItem?.type}
			/>
		</>
	);
}

export default App;