import axios, { AxiosRequestConfig } from "axios";

export class ApiCaller {
  constructor(
    private readonly serviceName: string,
    private readonly baseUrl: string,
  ) {
    this.serviceName = serviceName;
    this.baseUrl = baseUrl;
  }

  public static joinUrl(base: string, path: string) {
    return new URL(
      path.replace(/^\/+/, ""),
      `${base.replace(/\/+$/, "")}/`,
    ).toString();
  }

  public buildUrl(path: string): string {
    return ApiCaller.joinUrl(
      this.baseUrl,
      `${this.serviceName}/${path.replace(/^\/+/, "")}`,
    );
  }

  private async request<T>(config: AxiosRequestConfig): Promise<T> {
    try {
      const response = await axios<T>({
        ...config,
        baseURL: ApiCaller.joinUrl(this.baseUrl, this.serviceName),
        timeout: config.timeout ?? 10000,
      });

      return response.data;
    } catch (error) {
      if (axios.isAxiosError(error)) {
        const response = error.response?.data;
        if (
          response &&
          typeof response === "object" &&
          "error" in response &&
          typeof response.error === "string"
        ) {
          throw new Error(response.error, { cause: error });
        }
      }
      throw error;
    }
  }

  public get<T>(path: string, config?: AxiosRequestConfig): Promise<T> {
    return this.request<T>({
      method: "GET",
      url: path,
      ...config,
    });
  }

  public post<T>(
    path: string,
    data?: unknown,
    config?: AxiosRequestConfig,
  ): Promise<T> {
    return this.request<T>({
      method: "POST",
      url: path,
      data,
      ...config,
    });
  }

  public put<T>(
    path: string,
    data?: unknown,
    config?: AxiosRequestConfig,
  ): Promise<T> {
    return this.request<T>({
      method: "PUT",
      url: path,
      data,
      ...config,
    });
  }

  public delete<T>(path: string, config?: AxiosRequestConfig): Promise<T> {
    return this.request<T>({
      method: "DELETE",
      url: path,
      ...config,
    });
  }
}
