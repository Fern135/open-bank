import type { Metadata } from "next";
import { Archivo, Source_Serif_4 } from "next/font/google";

import "bootstrap/dist/css/bootstrap.min.css";
import "@fortawesome/fontawesome-free/css/all.min.css";
import "./globals.css";

import Config from "@config";

const archivo = Archivo({
  variable: "--font-archivo",
  subsets: ["latin"],
  axes: ["wdth"],
});

const sourceSerif = Source_Serif_4({
  variable: "--font-source-serif",
  subsets: ["latin"],
  style: ["normal", "italic"],
});

export const metadata: Metadata = {
  title: {
    default: Config.getAppName(),
    template: "%s | " + Config.getAppName(),
  },
  description: Config.getAppDescription(),
};

export default function RootLayout({ children }: LayoutProps<"/">) {
  return (
    <html lang="en" className={`${archivo.variable} ${sourceSerif.variable}`}>
      <body>
        <a className="visually-hidden-focusable" href="#main">
          Skip to content
        </a>
        {children}
      </body>
    </html>
  );
}
