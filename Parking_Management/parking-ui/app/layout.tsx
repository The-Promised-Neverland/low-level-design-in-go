import type { Metadata } from "next";
import { Barlow, Barlow_Condensed } from "next/font/google";
import "./globals.css";
const body = Barlow({ subsets: ["latin"], weight: ["400", "500", "600"], variable: "--f-body" });
const head = Barlow_Condensed({ subsets: ["latin"], weight: ["500", "600", "700"], variable: "--f-head" });
export const metadata: Metadata = { title: "Parking control room" };
export default function RootLayout({ children }: { children: React.ReactNode }) {
  return (<html lang="en"><body className={`${body.variable} ${head.variable}`}>{children}</body></html>);
}
