import { ChildProcess, spawn } from "child_process";
import { existsSync } from "fs";
import { app } from "electron";
import path from "path";
import { findAvailablePort } from "./port-utils";

let backendProcess: ChildProcess | undefined;
let backendPort: number | undefined;
let backendStartPromise: Promise<number> | undefined;

export async function startBackend(): Promise<number> {
  if (backendProcess && backendPort !== undefined) {
    return backendPort;
  }

  if (backendStartPromise) {
    return backendStartPromise;
  }

  backendStartPromise = startBackendProcess();

  try {
    return await backendStartPromise;
  } finally {
    backendStartPromise = undefined;
  }
}

async function startBackendProcess(): Promise<number> {
  const isDevelopment = !app.isPackaged;
  const backendDirectory = app.isPackaged
    ? path.join(process.resourcesPath, "backend")
    : path.join(app.getAppPath(), "assets");
  const backendNames =
    process.platform === "win32" ? ["lwn-server.exe", "lwn-server"] : ["lwn-server"];
  const backendPath = backendNames
    .map((backendName) => path.join(backendDirectory, backendName))
    .find((candidate) => existsSync(candidate));

  if (!backendPath) {
    throw new Error(
      `Local backend executable not found in ${backendDirectory}`,
    );
  }

  const port = await findAvailablePort();
  const childProcess = spawn(backendPath, ["-p", String(port)], {
    cwd: isDevelopment
      ? path.resolve(app.getAppPath(), "..", "backend")
      : undefined,
    env: {
      ...process.env,
      ...(isDevelopment ? { LWN_ENV: "development" } : {}),
    },
    stdio: "inherit",
    windowsHide: process.platform === "win32",
  });

  backendProcess = childProcess;
  backendPort = port;

  childProcess.once("error", () => {
    if (backendProcess === childProcess) {
      backendProcess = undefined;
      backendPort = undefined;
    }
  });

  childProcess.once("exit", () => {
    if (backendProcess === childProcess) {
      backendProcess = undefined;
      backendPort = undefined;
    }
  });

  try {
    await new Promise<void>((resolve, reject) => {
      const onSpawn = () => {
        cleanup();
        resolve();
      };
      const onError = (error: Error) => {
        cleanup();
        reject(error);
      };
      const cleanup = () => {
        childProcess.off("spawn", onSpawn);
        childProcess.off("error", onError);
      };

      childProcess.once("spawn", onSpawn);
      childProcess.once("error", onError);
    });
  } catch (error) {
    stopBackend();
    const message = error instanceof Error ? error.message : String(error);
    throw new Error(`Failed to start the local backend: ${message}`, {
      cause: error,
    });
  }

  console.log(`Backend running on port: ${port}`);

  return port;
}

export function stopBackend(): void {
  const processToStop = backendProcess;

  backendProcess = undefined;
  backendPort = undefined;

  if (processToStop && !processToStop.killed) {
    processToStop.kill();
  }
}

export async function waitForServer(
  port: number,
  timeoutMs = 10000,
): Promise<void> {
  const deadline = Date.now() + timeoutMs;

  while (Date.now() < deadline) {
    try {
      const response = await fetch(
        `http://127.0.0.1:${port}/api/app-info/status`,
        { signal: AbortSignal.timeout(1000) },
      );

      if (response.ok) {
        return;
      }
    } catch {
      // The backend may still be starting.
    }

    await new Promise((resolve) => setTimeout(resolve, 150));
  }

  throw new Error("Local backend did not become ready in time");
}
