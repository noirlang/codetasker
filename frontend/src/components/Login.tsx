import { useEffect, useRef, useState } from 'react';
import { Swiper, SwiperSlide } from 'swiper/react';
import { Autoplay, EffectCreative, Pagination } from 'swiper/modules';
import ScrollReveal from 'scrollreveal';
import {
  Code,
  GitPullRequest,
  Github,
  Kanban,
  Layers,
  RefreshCw,
  Sparkles,
} from 'lucide-react';

// Swiper CSS imports
import 'swiper/css';
import 'swiper/css/pagination';
import 'swiper/css/effect-creative';

// ── 3D Tilt Card Helper Component ──────────────────────────────────────────
function TiltCard({
  children,
  className = '',
  intensity = 15,
}: {
  children: React.ReactNode;
  className?: string;
  intensity?: number;
}) {
  const cardRef = useRef<HTMLDivElement>(null);
  const [tilt, setTilt] = useState({ x: 0, y: 0 });
  const [isHovered, setIsHovered] = useState(false);

  const handleMouseMove = (e: React.MouseEvent<HTMLDivElement>) => {
    if (!cardRef.current) return;
    const rect = cardRef.current.getBoundingClientRect();
    const width = rect.width;
    const height = rect.height;
    const mouseX = e.clientX - rect.left - width / 2;
    const mouseY = e.clientY - rect.top - height / 2;
    const rX = (mouseY / (height / 2)) * -intensity;
    const rY = (mouseX / (width / 2)) * intensity;
    setTilt({ x: rX, y: rY });
  };

  const handleMouseLeave = () => {
    setTilt({ x: 0, y: 0 });
    setIsHovered(false);
  };

  return (
    <div
      ref={cardRef}
      onMouseMove={handleMouseMove}
      onMouseEnter={() => setIsHovered(true)}
      onMouseLeave={handleMouseLeave}
      className={`transition-transform duration-200 ease-out ${className}`}
      style={{
        transform: `perspective(1000px) rotateX(${tilt.x}deg) rotateY(${tilt.y}deg) scale(${isHovered ? 1.02 : 1})`,
        transformStyle: 'preserve-3d',
      }}
    >
      <div style={{ transform: 'translateZ(40px)', transformStyle: 'preserve-3d' }}>
        {children}
      </div>
    </div>
  );
}

// ── Main Login Page ────────────────────────────────────────────────────────
export default function Login() {
  useEffect(() => {
    // Initialize ScrollReveal
    const sr = ScrollReveal({
      origin: 'bottom',
      distance: '60px',
      duration: 1000,
      delay: 100,
      opacity: 0,
      scale: 0.95,
      reset: false,
    });

    sr.reveal('.reveal-hero', { delay: 100 });
    sr.reveal('.reveal-cta', { delay: 350 });
    sr.reveal('.reveal-showcase', { delay: 200, origin: 'top' });
    sr.reveal('.reveal-step', { interval: 150 });
    sr.reveal('.reveal-features', { delay: 150 });
  }, []);

  return (
    <div className="relative min-h-screen bg-[#0a0a0a] text-white selection:bg-white selection:text-black">
      {/* ── Background Grid & Glowing Accent ────────────────────────────────── */}
      <div
        aria-hidden
        className="pointer-events-none absolute inset-0 z-0"
        style={{
          backgroundImage: `
            repeating-linear-gradient(
              0deg,
              transparent,
              transparent 49px,
              rgba(255, 255, 255, 0.02) 49px,
              rgba(255, 255, 255, 0.02) 50px
            ),
            repeating-linear-gradient(
              90deg,
              transparent,
              transparent 49px,
              rgba(255, 255, 255, 0.02) 49px,
              rgba(255, 255, 255, 0.02) 50px
            )
          `,
          opacity: 0.8,
        }}
      />
      
      {/* Subtle radial ambient lighting */}
      <div
        className="pointer-events-none absolute top-[-10%] left-[50%] h-[600px] w-[600px] -translate-x-[50%] rounded-full opacity-25 blur-[120px] transition-all duration-1000"
        style={{
          background: 'radial-gradient(circle, rgba(255,255,255,0.08) 0%, transparent 70%)',
        }}
      />

      {/* ── Section 1: Hero Section ─────────────────────────────────────────── */}
      <header className="relative z-10 mx-auto flex max-w-7xl flex-col items-center justify-between px-6 py-8 md:flex-row">
        <div className="flex items-center gap-2 select-none animate__animated animate__fadeIn">
          <img src="/logo.png" alt="CodeTasker" className="h-10 w-auto object-contain" />
        </div>
        <div className="mt-4 flex gap-4 md:mt-0 animate__animated animate__fadeIn">
          <a
            href="https://github.com/noirlang/codetasker"
            target="_blank"
            rel="noreferrer"
            className="flex items-center gap-1.5 text-xs text-[#a0a0a0] transition-colors hover:text-white"
          >
            <Github size={14} /> GitHub
          </a>
        </div>
      </header>

      <main className="relative z-10 mx-auto grid max-w-7xl grid-cols-1 items-center gap-16 px-6 pt-12 pb-24 lg:grid-cols-2">
        {/* Left Column: Heading & CTA */}
        <div className="flex flex-col items-start gap-8">
          <div className="reveal-hero flex items-center gap-2 rounded-full border border-white/10 bg-white/5 px-3 py-1 text-xs text-white/70">
            <Sparkles size={12} className="animate-pulse" />
            <span>Two-Way GitHub TODO Sync Engine</span>
          </div>

          <h1 className="reveal-hero text-left font-sans text-5xl font-extrabold leading-[1.1] tracking-tight text-white md:text-6xl">
            Turn your code comments into <span className="bg-gradient-to-r from-white via-white/80 to-white/40 bg-clip-text text-transparent">actionable tasks.</span>
          </h1>

          <p className="reveal-hero text-left text-lg text-[#a0a0a0] md:text-xl max-w-lg">
            Scans your codebase for annotations, maps them to interactive kanbans, and injects tasks directly back into your Git branch via automated Pull Requests.
          </p>

          <div className="reveal-cta w-full max-w-sm">
            <TiltCard intensity={10} className="w-full">
              <div className="flex w-full flex-col gap-4 rounded-lg border border-[#3a3a3a] bg-[#111111]/80 p-6 backdrop-blur-md">
                <p className="font-mono text-xs text-[#666666]">GET STARTED INSTANTLY</p>
                <a
                  href="/api/auth/github"
                  className="btn-primary w-full justify-center py-3.5 text-base font-semibold transition-all duration-300"
                  style={{ transformStyle: 'preserve-3d', transform: 'translateZ(10px)' }}
                >
                  <Github size={20} className="mr-1" />
                  Connect with GitHub
                </a>
                <p className="text-center text-xs text-[#666666]">
                  Safe integration. OAuth permissions required.
                </p>
              </div>
            </TiltCard>
          </div>
        </div>

        {/* Right Column: 3D Code Mockup Showcase */}
        <div className="reveal-showcase flex justify-center">
          <TiltCard intensity={15} className="w-full max-w-lg">
            <div className="relative overflow-hidden rounded-lg border border-[#2a2a2a] bg-[#111111] p-6 shadow-2xl">
              {/* Window Controls */}
              <div className="mb-4 flex items-center gap-1.5 border-b border-[#2a2a2a] pb-3">
                <span className="h-3.5 w-3.5 rounded-full bg-[#ff5f56]/20 border border-[#ff5f56]/30" />
                <span className="h-3.5 w-3.5 rounded-full bg-[#ffbd2e]/20 border border-[#ffbd2e]/30" />
                <span className="h-3.5 w-3.5 rounded-full bg-[#27c93f]/20 border border-[#27c93f]/30" />
                <span className="ml-3 font-mono text-[11px] text-[#666666]">
                  internal/service/task_service.go
                </span>
              </div>

              {/* Code Panel */}
              <pre className="overflow-x-auto font-mono text-xs text-[#a0a0a0] leading-relaxed">
                <code>
                  {`package service

func (s *TaskService) ProcessWebhook(...) {
`}
                  <span className="block border-l-2 border-white bg-white/5 px-2 py-0.5 font-bold text-white">
                    {`    // TODO: Verify webhook HMAC signature [BUG]`}
                  </span>
                  {`    if !validSignature {
        return ErrUnauthorized
    }

`}
                  <span className="block border-l-2 border-white/50 bg-white/5 px-2 py-0.5 font-semibold text-[#808080]">
                    {`    // FIXME: Fix token decryption routine`}
                  </span>
                  {`    decrypted, err := Decrypt(token)
}`}
                </code>
              </pre>

              {/* Connected Task Float Box */}
              <div
                className="absolute right-4 bottom-4 w-72 rounded border border-[#3a3a3a] bg-[#1a1a1a]/95 p-4 shadow-xl transition-all duration-300 hover:border-white"
                style={{ transform: 'translateZ(30px)' }}
              >
                <div className="mb-2 flex items-center justify-between">
                  <span className="tag text-[10px]">BUG</span>
                  <span className="font-mono text-[10px] text-[#666666]">Line 5</span>
                </div>
                <h4 className="font-mono text-xs font-semibold text-white">Verify webhook HMAC signature</h4>
                <div className="mt-3 flex items-center justify-between border-t border-[#2a2a2a] pt-2">
                  <span className="text-[10px] text-[#666666]">Commit: cb658cf</span>
                  <span className="flex items-center gap-1 text-[10px] text-white">
                    <span className="h-1.5 w-1.5 rounded-full bg-white animate-pulse" /> Open
                  </span>
                </div>
              </div>
            </div>
          </TiltCard>
        </div>
      </main>

      {/* ── Partner Section ────────────────────────────────────────────────── */}
      <section className="relative z-10 py-8 border-t border-[#2a2a2a]/40 bg-[#0d0d0d]">
        <div className="mx-auto max-w-7xl px-6 flex flex-col md:flex-row items-center justify-between gap-6">
          <p className="text-xs font-mono text-[#666666] tracking-wider uppercase">
            Enterprise Partner
          </p>
          <div className="flex items-center gap-4 opacity-50 hover:opacity-80 transition-opacity duration-200">
            <img src="/sirket.png" alt="Partner Logo" className="h-10 w-auto object-contain" />
          </div>
        </div>
      </section>

      {/* ── Section 2: Swiper 3D Feature Slider ─────────────────────────────── */}
      <section className="relative z-10 border-y border-[#2a2a2a] bg-[#111111]/30 py-24 backdrop-blur-md">
        <div className="mx-auto max-w-7xl px-6">
          <div className="reveal-features mb-16 text-center">
            <h2 className="text-3xl font-extrabold tracking-tight text-white sm:text-4xl">
              Engineered for developer workflows
            </h2>
            <p className="mx-auto mt-4 max-w-2xl text-base text-[#a0a0a0]">
              Automate task synchronization without leaving your terminal. Beautifully mapped visual workflows.
            </p>
          </div>

          <div className="reveal-features mx-auto max-w-4xl">
            <Swiper
              modules={[Autoplay, EffectCreative, Pagination]}
              grabCursor={true}
              effect={'creative'}
              creativeEffect={{
                prev: {
                  shadow: true,
                  translate: ['-20%', 0, -1],
                },
                next: {
                  translate: ['100%', 0, 0],
                },
              }}
              autoplay={{
                delay: 4000,
                disableOnInteraction: false,
              }}
              pagination={{ clickable: true }}
              className="mySwiper overflow-visible"
            >
              {/* Slide 1 */}
              <SwiperSlide className="py-4">
                <TiltCard intensity={5} className="w-full">
                  <div className="flex flex-col gap-6 rounded-lg border border-[#2a2a2a] bg-[#161616] p-8 shadow-lg md:flex-row md:items-center">
                    <div className="flex h-16 w-16 shrink-0 items-center justify-center rounded-lg border border-[#3a3a3a] bg-[#242424] text-white">
                      <Code size={32} />
                    </div>
                    <div className="flex-1">
                      <h3 className="font-mono text-xl font-bold text-white">Automatic TODO Parser</h3>
                      <p className="mt-2 text-[#a0a0a0]">
                        Runs a concurrent scanner written in Golang that listens to push webhook payloads, extracting comments matching patterns like TODO, FIXME, BUG, and HACK.
                      </p>
                    </div>
                  </div>
                </TiltCard>
              </SwiperSlide>

              {/* Slide 2 */}
              <SwiperSlide className="py-4">
                <TiltCard intensity={5} className="w-full">
                  <div className="flex flex-col gap-6 rounded-lg border border-[#2a2a2a] bg-[#161616] p-8 shadow-lg md:flex-row md:items-center">
                    <div className="flex h-16 w-16 shrink-0 items-center justify-center rounded-lg border border-[#3a3a3a] bg-[#242424] text-white">
                      <RefreshCw size={32} />
                    </div>
                    <div className="flex-1">
                      <h3 className="font-mono text-xl font-bold text-white">Real-time Webhook Sync</h3>
                      <p className="mt-2 text-[#a0a0a0]">
                        Whenever code is pushed to your remote GitHub repository, tasks are updated in real-time on your boards. Unreferenced tasks are automatically archived.
                      </p>
                    </div>
                  </div>
                </TiltCard>
              </SwiperSlide>

              {/* Slide 3 */}
              <SwiperSlide className="py-4">
                <TiltCard intensity={5} className="w-full">
                  <div className="flex flex-col gap-6 rounded-lg border border-[#2a2a2a] bg-[#161616] p-8 shadow-lg md:flex-row md:items-center">
                    <div className="flex h-16 w-16 shrink-0 items-center justify-center rounded-lg border border-[#3a3a3a] bg-[#242424] text-white">
                      <Kanban size={32} />
                    </div>
                    <div className="flex-1">
                      <h3 className="font-mono text-xl font-bold text-white">Minimalist Kanban Boards</h3>
                      <p className="mt-2 text-[#a0a0a0]">
                        Track issues visually. Move task cards between Open, In Progress, and Resolved columns with instant optimistic rendering that rolls back only on server errors.
                      </p>
                    </div>
                  </div>
                </TiltCard>
              </SwiperSlide>

              {/* Slide 4 */}
              <SwiperSlide className="py-4">
                <TiltCard intensity={5} className="w-full">
                  <div className="flex flex-col gap-6 rounded-lg border border-[#2a2a2a] bg-[#161616] p-8 shadow-lg md:flex-row md:items-center">
                    <div className="flex h-16 w-16 shrink-0 items-center justify-center rounded-lg border border-[#3a3a3a] bg-[#242424] text-white">
                      <GitPullRequest size={32} />
                    </div>
                    <div className="flex-1">
                      <h3 className="font-mono text-xl font-bold text-white">PR Injection Pipeline</h3>
                      <p className="mt-2 text-[#a0a0a0]">
                        Write a description directly inside the CodeTasker panel and push. CodeTasker will dynamically write the TODO annotation into your code and open a PR.
                      </p>
                    </div>
                  </div>
                </TiltCard>
              </SwiperSlide>
            </Swiper>
          </div>
        </div>
      </section>

      {/* ── Section 3: Interactive How it Works ─────────────────────────────── */}
      <section className="relative z-10 py-24">
        <div className="mx-auto max-w-7xl px-6">
          <div className="mb-20 text-center">
            <h2 className="text-3xl font-extrabold text-white">
              How{' '}
              <span style={{ fontFamily: "'Camiro', serif", letterSpacing: '0.05em' }} className="font-semibold">
                CodeTasker
              </span>{' '}
              works
            </h2>
            <p className="mt-4 text-[#a0a0a0]">Syncing your codebase annotation in 3 simple steps</p>
          </div>

          <div className="grid grid-cols-1 gap-12 md:grid-cols-3">
            {/* Step 1 */}
            <div className="reveal-step flex flex-col items-center text-center">
              <TiltCard intensity={12} className="mb-6 flex h-16 w-16 items-center justify-center rounded-full border border-white/20 bg-white/5 text-white">
                <span className="font-mono text-xl font-bold">1</span>
              </TiltCard>
              <h3 className="font-mono text-lg font-bold text-white">Annotate Code</h3>
              <p className="mt-2 text-sm text-[#a0a0a0] max-w-xs">
                Write standard comments inside your files: `// TODO: implement caching`. Push your commits.
              </p>
            </div>

            {/* Step 2 */}
            <div className="reveal-step flex flex-col items-center text-center">
              <TiltCard intensity={12} className="mb-6 flex h-16 w-16 items-center justify-center rounded-full border border-white/20 bg-white/5 text-white">
                <span className="font-mono text-xl font-bold">2</span>
              </TiltCard>
              <h3 className="font-mono text-lg font-bold text-white">Sync Dashboard</h3>
              <p className="mt-2 text-sm text-[#a0a0a0] max-w-xs">
                Webhooks instantly update your dashboard. See what needs attention, filtered by category and file paths.
              </p>
            </div>

            {/* Step 3 */}
            <div className="reveal-step flex flex-col items-center text-center">
              <TiltCard intensity={12} className="mb-6 flex h-16 w-16 items-center justify-center rounded-full border border-white/20 bg-white/5 text-white">
                <span className="font-mono text-xl font-bold">3</span>
              </TiltCard>
              <h3 className="font-mono text-lg font-bold text-white">Resolve & Pull</h3>
              <p className="mt-2 text-sm text-[#a0a0a0] max-w-xs">
                Change status on the board, or inject comments dynamically from the web console. Everything links back to pull requests.
              </p>
            </div>
          </div>
        </div>
      </section>

      {/* ── Section 4: Security ─────────────────────────────────────────────── */}
      <section className="relative z-10 border-t border-[#2a2a2a] bg-[#111111]/10 py-24 backdrop-blur-md">
        <div className="mx-auto max-w-7xl px-6">
          <div className="grid grid-cols-1 items-center gap-16 lg:grid-cols-2">
            <div>
              <h2 className="text-3xl font-extrabold text-white">Developer Security First</h2>
              <p className="mt-4 text-[#a0a0a0] leading-relaxed">
                CodeTasker takes authorization seriously. Access tokens are encrypted inside MongoDB using AES-256-GCM. Webhook communications are authenticated with HMAC-SHA256 signatures, shielding your API.
              </p>

              <div className="mt-8 flex flex-col gap-4">
                <div className="flex items-center gap-3">
                  <div className="flex h-8 w-8 items-center justify-center rounded border border-[#3a3a3a] bg-[#161616] text-white">
                    <Layers size={14} />
                  </div>
                  <span className="text-sm font-semibold text-white">AES-256-GCM Token Encryption</span>
                </div>
                <div className="flex items-center gap-3">
                  <div className="flex h-8 w-8 items-center justify-center rounded border border-[#3a3a3a] bg-[#161616] text-white">
                    <Layers size={14} />
                  </div>
                  <span className="text-sm font-semibold text-white">HMAC Webhook Signatures</span>
                </div>
              </div>
            </div>

            <div className="flex justify-center">
              <TiltCard intensity={8} className="w-full max-w-sm">
                <div className="rounded-lg border border-[#2a2a2a] bg-[#161616] p-8 text-center">
                  <div className="mx-auto mb-4 flex h-16 w-16 items-center justify-center rounded-full border border-white/20 bg-white/5 text-white">
                    <Layers size={24} />
                  </div>
                  <h3 className="font-mono text-lg font-bold text-white">Encrypted Datastore</h3>
                  <p className="mt-2 text-xs text-[#a0a0a0]">
                    Your code is read-only, and any modifications require explicitly signed git pipelines and authorized scopes.
                  </p>
                </div>
              </TiltCard>
            </div>
          </div>
        </div>
      </section>

      <footer className="relative z-10 border-t border-[#2a2a2a] py-8 text-center text-xs text-[#666666]">
        <p>
          © 2026{' '}
          <a
            href="https://noirlang.tr"
            target="_blank"
            rel="noopener noreferrer"
            className="hover:text-white transition-colors"
          >
            noirLang
          </a>
          . Sync your TODOs. Ship faster.
        </p>
      </footer>
    </div>
  );
}
