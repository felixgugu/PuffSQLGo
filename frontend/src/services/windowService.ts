import { isWails } from './api';
import {
  Quit,
  WindowIsMaximised,
  WindowMinimise,
  WindowToggleMaximise,
} from '../../wailsjs/runtime/runtime';

/**
 * 視窗控制服務：封裝 Wails runtime，避免 UI 元件直接相依原生橋接 API。
 *
 * 所有方法在瀏覽器 (Vite) 開發模式下皆為安全的無作用版本，因此元件可無條件呼叫。
 */

export const windowService = {
  /**
   * 目前平台是否使用「自繪標題列」。
   *
   * Wails 主視窗使用無邊框模式 (Frameless: true)，
   * 其餘平台仍保留系統原生視窗，因此不需要額外的視窗控制按鈕。
   */
  isCustomTitleBar(): boolean {
    if (!isWails()) return false;
    if (typeof navigator === 'undefined') return false;
    return /Windows/i.test(navigator.userAgent);
  },

  async isMaximized(): Promise<boolean> {
    if (!isWails()) return false;
    try {
      return await WindowIsMaximised();
    } catch (err) {
      console.warn('[PuffSQL] Unable to read window maximized state:', err);
      return false;
    }
  },

  async minimize(): Promise<void> {
    if (!isWails()) return;
    try {
      WindowMinimise();
    } catch (err) {
      console.warn('[PuffSQL] Unable to minimize window:', err);
    }
  },

  async toggleMaximize(): Promise<void> {
    if (!isWails()) return;
    try {
      WindowToggleMaximise();
    } catch (err) {
      console.warn('[PuffSQL] Unable to toggle window maximize state:', err);
    }
  },

  async close(): Promise<void> {
    if (!isWails()) return;
    try {
      Quit();
    } catch (err) {
      console.warn('[PuffSQL] Unable to close window:', err);
    }
  },

  /**
   * 訂閱視窗尺寸變化，用於同步「最大化 / 還原」按鈕圖示。
   * 回傳取消訂閱函式；瀏覽器開發模式回傳空函式。
   */
  async onResized(handler: () => void): Promise<() => void> {
    if (!isWails()) return () => {};
    const listener = () => handler();
    window.addEventListener('resize', listener);
    return () => window.removeEventListener('resize', listener);
  },
};

export default windowService;
