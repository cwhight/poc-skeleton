import type { Metadata, Viewport } from "next";
import { ideaConfig } from "../idea.config";
import "./globals.css";

export const metadata: Metadata = {
  title: `${ideaConfig.name} — ${ideaConfig.tagline}`,
  description: ideaConfig.subcopy,
};

export const viewport: Viewport = {
  width: "device-width",
  initialScale: 1,
};

export default function RootLayout({ children }: { children: React.ReactNode }) {
  return (
    <html lang="en">
      <body>{children}</body>
    </html>
  );
}
