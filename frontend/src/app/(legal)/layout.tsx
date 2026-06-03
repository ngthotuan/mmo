import Link from "next/link";
import { Zap } from "lucide-react";

export default function LegalLayout({
  children,
}: Readonly<{ children: React.ReactNode }>) {
  return (
    <div className="min-h-full bg-white text-gray-900">
      <header className="border-b border-gray-200">
        <div className="mx-auto flex max-w-3xl items-center justify-between px-6 py-4">
          <Link href="/" className="flex items-center gap-2 font-semibold">
            <Zap className="h-5 w-5 text-indigo-600" />
            <span>AutoContent</span>
          </Link>
          <nav className="flex gap-6 text-sm text-gray-600">
            <Link href="/terms" className="hover:text-gray-900">
              Terms
            </Link>
            <Link href="/privacy" className="hover:text-gray-900">
              Privacy
            </Link>
          </nav>
        </div>
      </header>
      <main className="mx-auto max-w-3xl px-6 py-12">{children}</main>
      <footer className="border-t border-gray-200">
        <div className="mx-auto max-w-3xl px-6 py-6 text-sm text-gray-500">
          © {new Date().getFullYear()} AutoContent. All rights reserved.
        </div>
      </footer>
    </div>
  );
}
