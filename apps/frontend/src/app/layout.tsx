import "../styles/main.scss";
import "@lwn-simulator/ui-components/styles.css";
import "material-symbols/index.css";
import { Metadata } from "next";
import { ProvidersWrapper } from "@/components";
import "react-toastify/ReactToastify.css";

export const metadata: Metadata = {
  title: "LWN Simulator",
  icons: ["/icon.png"],
};

export default function RootLayout({
  children,
}: Readonly<{
  children: React.ReactNode;
}>) {
  return (
    <html lang="en">
      <body>
        <ProvidersWrapper>{children}</ProvidersWrapper>
      </body>
    </html>
  );
}
