import { ApiCaller } from "./ApiCaller";

export class BaseService {
  protected readonly apiCaller: ApiCaller;

  constructor(serviceSlug: string, baseUrl: string) {
    this.apiCaller = new ApiCaller(serviceSlug, baseUrl);
  }
}
