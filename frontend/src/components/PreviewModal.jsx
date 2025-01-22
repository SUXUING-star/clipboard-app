export function PreviewModal({ isOpen, onClose, content, type }) {
  if (!isOpen) return null;

  return (
    <div 
      className="fixed inset-0 bg-black bg-opacity-50 z-50 flex items-center justify-center p-4"
      onClick={onClose}
    >
      <div 
        className="bg-white rounded-lg max-h-[90vh] max-w-[90vw] overflow-auto"
        onClick={e => e.stopPropagation()}
      >
        <div className="sticky top-0 bg-white border-b p-4 flex justify-between items-center">
          <h3 className="text-lg font-semibold">
            {type === 'text' ? '文本内容' : '图片预览'}
          </h3>
          <button
            onClick={onClose}
            className="text-gray-500 hover:text-gray-700"
          >
            关闭
          </button>
        </div>
        <div className="p-4">
          {type === 'text' ? (
            <div className="whitespace-pre-wrap break-words max-w-[800px]">
              {content}
            </div>
          ) : (
            <div className="flex items-center justify-center">
              <img
                src={content}
                alt="预览图片"
                className="max-w-full"
                style={{ maxHeight: 'calc(90vh - 120px)' }}
              />
            </div>
          )}
        </div>
      </div>
    </div>
  );
}