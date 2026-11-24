import "./globals.css";
import { Head } from "nextra/components";
import type { Metadata } from "next";
import type { ReactNode } from "react";
import { sitePath } from "@/utils";

export const metadata: Metadata = {
  title: "LWN Simulator Docs",
  description: "Documentation of LWN Simulator",
  icons: [`${sitePath}/favicon.png`],
};

export default async function RootLayout({
  children,
}: {
  children: ReactNode;
}) {
  return (
    <html lang="en" dir="ltr" suppressHydrationWarning>
        <Head />
      <body>
        {children}
      </body>
    </html>
  );
}
