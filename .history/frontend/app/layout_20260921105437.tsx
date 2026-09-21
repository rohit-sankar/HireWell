import type { Metadata } from "next";
import "./globals.css";
import { AuthProvider } from "@/providers/auth-provider";

export const metadata: Metadata = {
  title: "Hirewell — AI resume parsing & candidate matching",
  description:
    "Hirewell parses resumes into structured profiles, matches candidates against real job requirements, and runs grounded AI interviews based only on skills your job actually requires.",
};

export default function RootLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  return (
    <html lang="en" data-scroll-behavior="smooth">
      <body>
        <AuthProvider>{children}</AuthProvider>
      </body>
    </html>
  );
}
