import Link from "next/link";
import { Button } from "@/components/ui/button";

export default function HomePage() {
  return (
    <div className="relative flex min-h-full flex-1 flex-col unicore-grid overflow-hidden">
      <div
        aria-hidden
        className="pointer-events-none absolute inset-0 bg-[radial-gradient(circle_at_70%_20%,color-mix(in_srgb,var(--teal)_12%,transparent),transparent_45%)]"
      />

      <header className="relative z-10 flex items-center justify-between px-6 py-5 md:px-10">
        <span className="sr-only">Unicore</span>
        <div className="flex items-center gap-3">
          <span className="inline-block size-2.5 rounded-full bg-teal" />
          <span className="text-sm font-medium tracking-wide text-teal-soft">
            Campus ERP
          </span>
        </div>
        <nav className="flex items-center gap-2">
          <Button variant="ghost" size="sm" render={<Link href="/login" />}>
            Sign in
          </Button>
        </nav>
      </header>

      <main className="relative z-10 flex flex-1 flex-col justify-center px-6 pb-20 pt-8 md:px-10 md:pb-28">
        <div className="mx-auto w-full max-w-3xl">
          <p className="animate-fade-up text-[clamp(3.5rem,12vw,7.5rem)] font-semibold leading-[0.9] tracking-tight text-ink">
            Unicore
          </p>
          <h1 className="animate-fade-up-delay mt-6 max-w-xl text-xl font-medium leading-snug text-ink/90 md:text-2xl">
            One operating system for institutes — results, fees, attendance, and
            campus life.
          </h1>
          <p className="animate-fade-up-delay-2 mt-4 max-w-md text-base text-muted-foreground">
            Multi-tenant by design. Your institute on its own subdomain, with
            roles for students, faculty, and admins.
          </p>
          <div className="animate-fade-up-delay-2 mt-10 flex flex-wrap items-center gap-3">
            <Button size="lg" render={<Link href="/login" />}>
              Sign in
            </Button>
            <Button
              size="lg"
              variant="outline"
              render={<Link href="/signup" />}
            >
              Register your institute
            </Button>
          </div>
        </div>
      </main>

      <footer className="relative z-10 border-t border-border/60 px-6 py-4 text-xs text-muted-foreground md:px-10">
        Multi-tenant college ERP · Use your institute subdomain to sign in
      </footer>
    </div>
  );
}
