// useErrorHandler 组合式函数：封装 ant-design-vue 的 message 与 modal，
// 提供统一的错误处理、成功提示与确认对话框接口。

import { message, Modal } from 'ant-design-vue';

export interface ErrorHandler {
  handleError: (error: unknown) => void;
  showError: (msg: string) => void;
  showSuccess: (msg: string) => void;
  confirmAction: (msg: string, title?: string) => Promise<boolean>;
}

export function useErrorHandler(): ErrorHandler {
  function handleError(error: unknown): void {
    if (error instanceof Error) {
      message.error(error.message);
    } else if (typeof error === 'string') {
      message.error(error);
    } else {
      message.error('发生未知错误，请稍后重试');
    }
  }

  function showError(msg: string): void {
    message.error(msg);
  }

  function showSuccess(msg: string): void {
    message.success(msg);
  }

  function confirmAction(msg: string, title = '确认操作'): Promise<boolean> {
    return new Promise((resolve) => {
      Modal.confirm({
        title,
        content: msg,
        okText: '确认',
        cancelText: '取消',
        centered: true,
        onOk: () => resolve(true),
        onCancel: () => resolve(false),
      });
    });
  }

  return {
    handleError,
    showError,
    showSuccess,
    confirmAction,
  };
}