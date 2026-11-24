import { sitePath } from "@/utils";

export default function NotFound() {
  return (
    <main className="mx-auto flex min-h-screen max-w-2xl flex-col items-center justify-center gap-4 px-6 text-center">
      <h1 className="text-3xl font-semibold">Page not found</h1>
      <p className="text-gray-600 dark:text-gray-400">
        The requested documentation page does not exist.
      </p>
      <a className="nx-text-primary-600 underline" href={`${sitePath}/en`}>
        Open the English documentation
      </a>
    </main>
  );
}
