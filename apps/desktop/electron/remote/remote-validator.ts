type RemoteAppInfo = {
  app?: unknown;
  version?: unknown;
};

async function getRemoteAppInfo(url: string): Promise<RemoteAppInfo> {
  const remoteUrl = new URL(url);

  if (!["http:", "https:"].includes(remoteUrl.protocol)) {
    throw new Error("Only HTTP and HTTPS servers are supported");
  }

  const response = await fetch(new URL("/api/app-info/info", remoteUrl), {
    signal: AbortSignal.timeout(5000),
  });

  if (!response.ok) {
    throw new Error(`Request failed with status ${response.status}`);
  }

  return (await response.json()) as RemoteAppInfo;
}

export async function validateRemote(url: string) {
  try {
    const info = await getRemoteAppInfo(url);

    if (info.app !== "lwn-simulator") {
      throw new Error("Server isn't a LWN Simulator instance");
    }

    const version = process.env.APP_ENV || "dev";

    if (version !== "dev" && info.version !== version) {
      throw new Error("Incompatible version");
    }

    return info;
  } catch (err) {
    if (
      err instanceof Error &&
      [
        "Server isn't a LWN Simulator instance",
        "Incompatible version",
        "Only HTTP and HTTPS servers are supported",
      ].includes(err.message)
    ) {
      throw err;
    }

    throw new Error("Server unreachable", {
      cause: err,
    });
  }
}
