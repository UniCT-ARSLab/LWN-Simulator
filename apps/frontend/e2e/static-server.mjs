import { createReadStream, promises as fs } from "node:fs";
import { createServer } from "node:http";
import { extname, join, relative, resolve } from "node:path";
import { fileURLToPath } from "node:url";

const rootDirectory = resolve(
  fileURLToPath(new URL("../out", import.meta.url)),
);
const port = Number(process.env.PLAYWRIGHT_PORT ?? 3000);

const contentTypes = {
  ".css": "text/css; charset=utf-8",
  ".html": "text/html; charset=utf-8",
  ".js": "text/javascript; charset=utf-8",
  ".json": "application/json; charset=utf-8",
  ".svg": "image/svg+xml",
  ".txt": "text/plain; charset=utf-8",
  ".woff2": "font/woff2",
};

const sendNotFound = (response) => {
  response.writeHead(404, { "content-type": "text/plain; charset=utf-8" });
  response.end("Not found");
};

const server = createServer(async (request, response) => {
  const requestPath = decodeURIComponent(new URL(request.url ?? "/", "http://localhost").pathname);
  const candidatePath = resolve(rootDirectory, `.${requestPath}`);

  if (!relative(rootDirectory, candidatePath).startsWith("..")) {
    try {
      const stats = await fs.stat(candidatePath);
      if (stats.isDirectory()) {
        if (!requestPath.endsWith("/")) {
          response.writeHead(301, { location: `${requestPath}/${new URL(request.url ?? "/", "http://localhost").search}` });
          response.end();
          return;
        }

        const indexPath = join(candidatePath, "index.html");
        await fs.access(indexPath);
        response.writeHead(200, {
          "content-type": "text/html; charset=utf-8",
        });
        createReadStream(indexPath).pipe(response);
        return;
      }

      response.writeHead(200, {
        "content-type": contentTypes[extname(candidatePath)] ?? "application/octet-stream",
      });
      createReadStream(candidatePath).pipe(response);
      return;
    } catch {
      sendNotFound(response);
      return;
    }
  }

  sendNotFound(response);
});

server.listen(port, "127.0.0.1", () => {
  console.log(`Serving ${rootDirectory} on http://127.0.0.1:${port}`);
});
