import { Footer, Layout, Navbar } from "nextra-theme-docs";
import { getPageMap } from "nextra/page-map";
import Image from "next/image";
import type { ReactNode } from "react";
import { sitePath } from "@/utils";

const footer = <Footer>MIT {new Date().getFullYear()} © LWN Simulator.</Footer>;

export default async function LocaleLayout({
  children,
  params,
}: {
  children: ReactNode;
  params: Promise<{ locale: string }>;
}) {
  const { locale } = await params;

  return (
    <Layout
      navbar={
        <Navbar
          logo={
            <div className="flex items-center gap-2">
              <Image
                src={`${sitePath}/favicon.png`}
                alt="logo"
                width={50}
                height={50}
                className="w-12 h-12"
              />
              <h1 className="text-2xl">LWN Simulator</h1>
            </div>
          }
          projectLink="https://github.com/Giuseppe-Compagnone/LWN-Simulator"
        />
      }
      pageMap={await getPageMap(`/${locale}`)}
      i18n={[
        { locale: "en", name: "🇬🇧 English" },
        { locale: "it", name: "🇮🇹 Italiano" },
        { locale: "es", name: "🇪🇸 Español" },
        { locale: "fr", name: "🇫🇷 Français" },
        { locale: "de", name: "🇩🇪 Deutsch" },
      ]}
      docsRepositoryBase="https://github.com/Giuseppe-Compagnone/LWN-Simulator"
      editLink="Edit this page on GitHub"
      sidebar={{ defaultMenuCollapseLevel: 1, autoCollapse: true }}
      footer={footer}
    >
      {children}
    </Layout>
  );
}
