import { ToastOptions, toast } from "react-toastify";

export class NotificationHandler {
  /**
   * Class instance to Follow singleton design
   */
  private static _instance: NotificationHandler;

  /**
   * Get instance method or create if not exists
   */
  public static get instance(): NotificationHandler {
    if (!this._instance) {
      this._instance = new NotificationHandler();
    }

    return this._instance;
  }

  /**
   * Toastify common config
   */
  private _config: ToastOptions = {
    position: "bottom-right",
    autoClose: 8000,
    hideProgressBar: true,
    closeOnClick: true,
    pauseOnHover: true,
    draggable: false,
    progress: undefined,
    theme: "dark",
    className: "custom-toast",
  };

  /**
   * Success Notification
   *
   * @param text Notification Text
   */
  public success(text: string): void {
    toast.success(text, this._config);
  }

  /**
   * Error Notification
   *
   * @param text Notification Text
   */
  public error(text: string): void {
    toast.error(text, this._config);
  }

  /**
   * Warning Notification
   *
   * @param text Notification Text
   */
  public warning(text: string): void {
    toast.warning(text, this._config);
  }

  private constructor() {}
}
