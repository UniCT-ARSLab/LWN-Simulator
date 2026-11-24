import net from "net";

export function isPortAvailable(port: number): Promise<boolean> {
  return new Promise<boolean>((resolve) => {
    const server = net.createServer();
    let settled = false;

    const finish = (available: boolean) => {
      if (settled) {
        return;
      }

      settled = true;
      resolve(available);
    };

    server.once("error", () => finish(false));

    server.once("listening", () => {
      server.close(() => finish(true));
    });

    server.listen(port, "127.0.0.1");
  });
}

export async function findAvailablePort(
  startPort = 8080,
  endPort = 65535,
): Promise<number> {
  let port = startPort;

  while (port <= endPort && !(await isPortAvailable(port))) {
    port++;
  }

  if (port > endPort) {
    throw new Error("No available local port was found");
  }

  return port;
}
