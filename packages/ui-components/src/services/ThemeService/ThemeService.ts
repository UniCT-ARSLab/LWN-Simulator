import storage from "@/utils/storage";
import { Theme } from "./ThemeService.types";

/**
 * Service responsible for managing the application's theme.
 *
 * The service uses a singleton pattern and determines the default theme
 * from the persisted user preference or, when unavailable, from the
 * system color scheme preference.
 */
export class ThemeService {
  private static _instance: ThemeService | null = null;

  /**
   * Returns the singleton instance of the theme service.
   */
  public static get instance() {
    if (!this._instance) {
      this._instance = new ThemeService();
    }

    return this._instance;
  }

  /**
   * Determines the default theme for the application.
   *
   * The persisted theme preference is used when available. Otherwise,
   * the system color scheme preference is checked through the
   * `prefers-color-scheme` media query.
   *
   * @returns The persisted theme, or the system-preferred theme when
   * no persisted preference is available.
   */
  getDefaultTheme = async (): Promise<Theme> => {
    const cached = await storage.get("theme");

    if (cached) {
      return cached as Theme;
    }

    const prefersDark = window.matchMedia(
      "(prefers-color-scheme: dark)",
    ).matches;

    return prefersDark ? Theme.Dark : Theme.Light;
  };

  private constructor() {}
}
