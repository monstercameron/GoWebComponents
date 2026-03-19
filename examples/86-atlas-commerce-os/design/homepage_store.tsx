import React, { useEffect, useMemo, useState } from 'react'
import { AnimatePresence, motion } from 'framer-motion'
import { ArrowRight, ChevronDown, ChevronLeft, ChevronRight, Menu, Package, Search, ShoppingCart, Star, X } from 'lucide-react'

const fmt = (n: number) => new Intl.NumberFormat('en-US', { style: 'currency', currency: 'USD' }).format(n)
const cls = (...a: (string | false | undefined)[]) => a.filter(Boolean).join(' ')

type P = {
  id: number
  slug: string
  name: string
  sub: string
  cat: string
  price: number
  cmp: number
  stock: number
  feat: boolean
  sale: boolean
  rate: number
  img: string
  gallery: string[]
  desc: string
  tags: string[]
}

type C = { id: number; name: string; price: number; img: string; qty: number }
type Crumb = { t: string; k?: string }
type F = { n: string; e: string; a: string; c: string; s: string; z: string; card: string; exp: string; cvv: string }

const cats = ['Audio', 'Desk', 'Lighting', 'Travel', 'Wearables']

const prods0: P[] = [
  { id: 1, slug: 'aero-headphones', name: 'Aero Headphones', sub: 'Wireless spatial audio', cat: 'Audio', price: 249, cmp: 319, stock: 18, feat: true, sale: true, rate: 4.8, img: 'https://images.unsplash.com/photo-1505740420928-5e560c06d30e?auto=format&fit=crop&w=1200&q=80', gallery: ['https://images.unsplash.com/photo-1505740420928-5e560c06d30e?auto=format&fit=crop&w=1200&q=80', 'https://images.unsplash.com/photo-1546435770-a3e426bf472b?auto=format&fit=crop&w=1200&q=80'], desc: 'Clean sound, soft pads, long battery life.', tags: ['wireless', 'premium'] },
  { id: 2, slug: 'halo-lamp', name: 'Halo Lamp', sub: 'Ambient desk glow', cat: 'Lighting', price: 129, cmp: 159, stock: 9, feat: true, sale: false, rate: 4.6, img: 'https://images.unsplash.com/photo-1507473885765-e6ed057f782c?auto=format&fit=crop&w=1200&q=80', gallery: ['https://images.unsplash.com/photo-1507473885765-e6ed057f782c?auto=format&fit=crop&w=1200&q=80'], desc: 'Soft light with a sculpted profile for modern desks.', tags: ['ambient', 'desk'] },
  { id: 3, slug: 'atlas-pack', name: 'Atlas Pack', sub: 'Minimal travel bag', cat: 'Travel', price: 189, cmp: 229, stock: 31, feat: false, sale: true, rate: 4.7, img: 'https://images.unsplash.com/photo-1542291026-7eec264c27ff?auto=format&fit=crop&w=1200&q=80', gallery: ['https://images.unsplash.com/photo-1542291026-7eec264c27ff?auto=format&fit=crop&w=1200&q=80'], desc: 'Balanced storage with premium fabric and clean lines.', tags: ['travel', 'carry'] },
  { id: 4, slug: 'drift-keyboard', name: 'Drift Keyboard', sub: 'Low profile mechanical feel', cat: 'Desk', price: 139, cmp: 169, stock: 24, feat: true, sale: false, rate: 4.5, img: 'https://images.unsplash.com/photo-1511467687858-23d96c32e4ae?auto=format&fit=crop&w=1200&q=80', gallery: ['https://images.unsplash.com/photo-1511467687858-23d96c32e4ae?auto=format&fit=crop&w=1200&q=80'], desc: 'Quiet switches and compact layout for focused work.', tags: ['desk', 'input'] },
  { id: 12, slug: 'arc-case', name: 'Arc Case', sub: 'Structured hard shell carry', cat: 'Travel', price: 219, cmp: 269, stock: 7, feat: true, sale: false, rate: 4.7, img: 'https://images.unsplash.com/photo-1524504388940-b1c1722653e1?auto=format&fit=crop&w=1200&q=80', gallery: ['https://images.unsplash.com/photo-1524504388940-b1c1722653e1?auto=format&fit=crop&w=1200&q=80'], desc: 'Hard shell structure with elegant travel-ready details.', tags: ['case', 'travel'] },
]

export default function App() {
  const [route, setRoute] = useState('/home')
  const [menu, setMenu] = useState(false)
  const [cartOpen, setCartOpen] = useState(false)
  const [cart, setCart] = useState<C[]>([])
  const [prods] = useState<P[]>(prods0)
  const [pick, setPick] = useState<P>(prods0[0])
  const [note, setNote] = useState('')

  const nav = (p: string) => {
    setRoute(p)
    setMenu(false)
    if (typeof window !== 'undefined') window.scrollTo({ top: 0, behavior: 'smooth' })
  }

  const goItem = (p: P) => {
    setPick(p)
    nav('/store/item')
  }

  const add = (p: P, n = 1) => {
    setCartOpen(true)
    setNote(`${p.name} added to cart`)
    setCart(c => {
      const i = c.findIndex(x => x.id === p.id)
      if (i >= 0) {
        const next = [...c]
        next[i] = { ...next[i], qty: next[i].qty + n }
        return next
      }
      return [...c, { id: p.id, name: p.name, price: p.price, img: p.img, qty: n }]
    })
  }

  const qty = (id: number, d: number) => setCart(c => c.map(x => x.id === id ? { ...x, qty: Math.max(1, x.qty + d) } : x))
  const rm = (id: number) => setCart(c => c.filter(x => x.id !== id))
  const sub = cart.reduce((a, b) => a + b.price * b.qty, 0)
  const tax = sub * 0.07
  const ship = cart.length ? (sub > 200 ? 0 : 18) : 0
  const tot = sub + tax + ship

  return (
    <div className="min-h-screen bg-zinc-950 text-zinc-100">
      <Bg />
      <Top nav={nav} route={route} cartN={cart.length} menu={menu} setMenu={setMenu} setCartOpen={setCartOpen} />
      <main className="mx-auto max-w-7xl px-4 pb-16 pt-24 sm:px-6 lg:px-8">
        {route === '/home' && <Home nav={nav} prods={prods} add={add} setPick={goItem} />}
        {route === '/store' && <Store prods={prods} add={add} nav={nav} setPick={goItem} />}
        {route === '/store/item' && <Item p={pick} add={add} setPick={goItem} prods={prods} nav={nav} />}
        {route === '/checkout' && <Checkout cart={cart} sub={sub} tax={tax} ship={ship} tot={tot} nav={nav} setCart={setCart} />}
      </main>
      <Cart open={cartOpen} setOpen={setCartOpen} cart={cart} qty={qty} rm={rm} sub={sub} tax={tax} ship={ship} tot={tot} nav={nav} />
      <Toast note={note} setNote={setNote} />
    </div>
  )
}

function Bg() {
  return (
    <div className="pointer-events-none fixed inset-0 overflow-hidden">
      <div className="absolute left-[-10%] top-[-5%] h-72 w-72 rounded-full bg-indigo-500/20 blur-3xl" />
      <div className="absolute right-[-5%] top-[10%] h-80 w-80 rounded-full bg-cyan-500/15 blur-3xl" />
      <div className="absolute bottom-[-10%] left-[20%] h-96 w-96 rounded-full bg-emerald-500/10 blur-3xl" />
      <div className="absolute inset-0 bg-[radial-gradient(circle_at_top,rgba(255,255,255,0.06),transparent_30%),linear-gradient(to_bottom,rgba(24,24,27,0.88),rgba(9,9,11,1))]" />
    </div>
  )
}

function Top({ nav, route, cartN, menu, setMenu, setCartOpen }: { nav: (p: string) => void; route: string; cartN: number; menu: boolean; setMenu: React.Dispatch<React.SetStateAction<boolean>>; setCartOpen: React.Dispatch<React.SetStateAction<boolean>> }) {
  const links = [
    { k: '/home', t: 'Home' },
    { k: '/store', t: 'Store' },
  ]
  return (
    <>
      <div className="fixed inset-x-0 top-0 z-50 border-b border-white/10 bg-white/5 backdrop-blur-xl">
        <div className="mx-auto flex max-w-7xl items-center justify-between px-4 py-3 sm:px-6 lg:px-8">
          <button onClick={() => nav('/home')} className="flex items-center gap-3">
            <div className="grid h-10 w-10 place-items-center rounded-2xl border border-white/15 bg-white/10 shadow-2xl shadow-black/30">
              <Package className="h-5 w-5" />
            </div>
            <div className="text-left">
              <div className="text-sm text-zinc-400">Modern Commerce</div>
              <div className="text-base font-semibold">Atelier Mock</div>
            </div>
          </button>
          <div className="hidden items-center gap-2 md:flex">
            {links.map(x => (
              <button key={x.k} onClick={() => nav(x.k)} className={cls('rounded-2xl px-4 py-2 text-sm transition', route === x.k ? 'border border-white/15 bg-white/10 shadow-lg shadow-black/20' : 'text-zinc-300 hover:bg-white/5 hover:text-white')}>
                {x.t}
              </button>
            ))}
          </div>
          <div className="flex items-center gap-2">
            <button onClick={() => setCartOpen(true)} className="relative rounded-2xl border border-white/15 bg-white/10 p-3 shadow-lg shadow-black/20 transition hover:bg-white/15">
              <ShoppingCart className="h-5 w-5" />
              {cartN > 0 && <span className="absolute -right-1 -top-1 grid h-5 min-w-5 place-items-center rounded-full bg-white px-1 text-[11px] font-semibold text-zinc-900">{cartN}</span>}
            </button>
            <button onClick={() => setMenu(v => !v)} className="rounded-2xl border border-white/15 bg-white/10 p-3 md:hidden">
              <Menu className="h-5 w-5" />
            </button>
          </div>
        </div>
      </div>
      <AnimatePresence>
        {menu && (
          <motion.div initial={{ opacity: 0, y: -8 }} animate={{ opacity: 1, y: 0 }} exit={{ opacity: 0, y: -8 }} className="fixed inset-x-4 top-20 z-40 rounded-3xl border border-white/10 bg-zinc-900/80 p-3 backdrop-blur-2xl md:hidden">
            {links.map(x => (
              <button key={x.k} onClick={() => nav(x.k)} className="flex w-full items-center justify-between rounded-2xl px-4 py-3 text-left text-zinc-200 hover:bg-white/5">
                {x.t}
                <ArrowRight className="h-4 w-4" />
              </button>
            ))}
          </motion.div>
        )}
      </AnimatePresence>
    </>
  )
}

function Glass({ className = '', children }: { className?: string; children: React.ReactNode }) {
  return <div className={cls('rounded-[28px] border border-white/10 bg-white/5 backdrop-blur-xl shadow-2xl shadow-black/20', className)}>{children}</div>
}

function Btn({ className = '', onClick, children, kind = 'a', type = 'button' }: { className?: string; onClick?: () => void; children: React.ReactNode; kind?: 'a' | 'b'; type?: 'button' | 'submit' }) {
  return <button type={type} onClick={onClick} className={cls('rounded-2xl px-4 py-3 text-sm font-medium transition active:scale-[0.98]', kind === 'a' ? 'bg-white text-zinc-950 shadow-xl shadow-black/20 hover:bg-zinc-200' : 'border border-white/15 bg-white/10 text-white hover:bg-white/15', className)}>{children}</button>
}

function Sec({ eye, title, sub, right }: { eye: string; title: string; sub?: string; right?: React.ReactNode }) {
  return (
    <div className="mb-6 flex items-end justify-between gap-4">
      <div>
        <div className="mb-2 text-xs uppercase tracking-[0.24em] text-zinc-400">{eye}</div>
        <h2 className="text-2xl font-semibold tracking-tight sm:text-3xl">{title}</h2>
        {sub && <p className="mt-2 max-w-2xl text-sm leading-6 text-zinc-400 sm:text-base">{sub}</p>}
      </div>
      {right}
    </div>
  )
}

function Badge({ children, tone = 'z' }: { children: React.ReactNode; tone?: 'z' | 'g' | 'b' | 'y' | 'r' }) {
  const m = {
    z: 'border-white/10 bg-white/5 text-zinc-300',
    g: 'border-emerald-400/20 bg-emerald-400/10 text-emerald-200',
    b: 'border-cyan-400/20 bg-cyan-400/10 text-cyan-200',
    y: 'border-amber-400/20 bg-amber-400/10 text-amber-200',
    r: 'border-rose-400/20 bg-rose-400/10 text-rose-200',
  }
  return <span className={cls('inline-flex items-center rounded-full border px-3 py-1 text-xs', m[tone])}>{children}</span>
}

function Crumbs({ nav, items }: { nav: (p: string) => void; items: Crumb[] }) {
  return (
    <div className="mb-5 flex flex-wrap items-center gap-2 text-sm text-zinc-500">
      {items.map((x, i) => (
        <React.Fragment key={`${x.t}-${i}`}>
          {i > 0 && <ChevronRight className="h-3.5 w-3.5 text-zinc-700" />}
          {x.k ? <button onClick={() => nav(x.k)} className="transition hover:text-white">{x.t}</button> : <span className="text-zinc-300">{x.t}</span>}
        </React.Fragment>
      ))}
    </div>
  )
}

function Home({ nav, prods, add, setPick }: { nav: (p: string) => void; prods: P[]; add: (p: P, n?: number) => void; setPick: (p: P) => void }) {
  const feat = prods.filter(x => x.feat).slice(0, 4)
  const sale = prods.filter(x => x.sale).slice(0, 3)
  const byCat = cats.map(cat => ({ cat, img: prods.find(x => x.cat === cat)?.img || prods[0].img }))
  return (
    <div className="space-y-10">
      <Glass className="relative overflow-hidden p-6 sm:p-8 lg:p-10">
        <div className="absolute inset-y-0 right-0 hidden w-1/2 bg-gradient-to-l from-white/10 to-transparent lg:block" />
        <div className="grid items-center gap-8 lg:grid-cols-2">
          <div>
            <Badge tone="b">Spring event · up to 35% off</Badge>
            <h1 className="mt-4 text-4xl font-semibold tracking-tight sm:text-5xl lg:text-6xl">Premium objects for calmer, cleaner daily life.</h1>
            <p className="mt-4 max-w-xl text-base leading-7 text-zinc-300 sm:text-lg">A modern storefront mock with elevated merchandising, responsive browsing, refined cart flows, and a polished simulated checkout.</p>
            <div className="mt-8 flex flex-wrap gap-3">
              <Btn onClick={() => nav('/store')}>Shop collection</Btn>
              <Btn kind="b" onClick={() => nav('/store')}>Explore sale</Btn>
            </div>
            <div className="mt-8 grid grid-cols-3 gap-3">
              {[
                ['12', 'Curated items'],
                ['5', 'Categories'],
                ['Free', 'Shipping $200+'],
              ].map(x => (
                <Glass key={x[1]} className="p-4">
                  <div className="text-2xl font-semibold">{x[0]}</div>
                  <div className="mt-1 text-sm text-zinc-400">{x[1]}</div>
                </Glass>
              ))}
            </div>
          </div>
          <div className="grid gap-4 sm:grid-cols-2">
            <Glass className="overflow-hidden p-0 sm:col-span-2">
              <div className="grid sm:grid-cols-[1.1fr_.9fr]">
                <div className="p-6 sm:p-7">
                  <Badge tone="y">Limited drop</Badge>
                  <div className="mt-4 text-3xl font-semibold">Design-led essentials with soft form and premium finish.</div>
                  <p className="mt-3 text-sm leading-6 text-zinc-400">Audio, desk, travel, and wearable goods presented in a clean premium interface.</p>
                  <div className="mt-6"><Btn onClick={() => nav('/store')}>Browse now</Btn></div>
                </div>
                <img src={feat[0]?.img} className="h-full min-h-64 w-full object-cover" />
              </div>
            </Glass>
            <Glass className="p-5">
              <div className="mb-3 flex items-center justify-between"><Badge tone="g">Featured</Badge><Star className="h-4 w-4 text-zinc-400" /></div>
              <div className="text-xl font-medium">Curated product cards</div>
              <p className="mt-2 text-sm leading-6 text-zinc-400">Soft surfaces, subtle borders, premium spacing, and gentle motion.</p>
            </Glass>
            <Glass className="p-5">
              <div className="mb-3 flex items-center justify-between"><Badge tone="b">Checkout</Badge><ArrowRight className="h-4 w-4 text-zinc-400" /></div>
              <div className="text-xl font-medium">Fast simulated purchase flow</div>
              <p className="mt-2 text-sm leading-6 text-zinc-400">Product page, cart drawer, checkout form, and clean order confirmation.</p>
            </Glass>
          </div>
        </div>
      </Glass>

      <section>
        <Sec eye="Categories" title="Shop by category" sub="Clear paths into the collection with strong mobile and desktop composition." />
        <div className="grid gap-4 sm:grid-cols-2 xl:grid-cols-5">
          {byCat.map(x => (
            <button key={x.cat} onClick={() => nav('/store')} className="group text-left">
              <Glass className="overflow-hidden p-0">
                <div className="relative aspect-[4/5] overflow-hidden">
                  <img src={x.img} className="h-full w-full object-cover transition duration-500 group-hover:scale-105" />
                  <div className="absolute inset-0 bg-gradient-to-t from-black/70 to-transparent" />
                  <div className="absolute bottom-0 left-0 right-0 p-4">
                    <div className="text-lg font-medium">{x.cat}</div>
                    <div className="mt-1 text-sm text-zinc-300">Explore collection</div>
                  </div>
                </div>
              </Glass>
            </button>
          ))}
        </div>
      </section>

      <section>
        <Sec eye="Featured" title="Best of the collection" sub="Handpicked items with clean aesthetics and premium positioning." right={<Btn kind="b" onClick={() => nav('/store')}>View all</Btn>} />
        <div className="grid gap-4 sm:grid-cols-2 xl:grid-cols-4">
          {feat.map(p => <ProdCard key={p.id} p={p} add={add} setPick={setPick} />)}
        </div>
      </section>

      <section>
        <Sec eye="Promotions" title="Limited offers" sub="A polished promo layout with restrained color accents and high-end merchandising." />
        <div className="grid gap-4 lg:grid-cols-[1.4fr_.9fr]">
          <Glass className="overflow-hidden p-6">
            <div className="grid gap-6 lg:grid-cols-[1.1fr_.9fr]">
              <div>
                <Badge tone="y">Save on essentials</Badge>
                <div className="mt-4 text-3xl font-semibold">Build a calmer, cleaner setup.</div>
                <p className="mt-3 text-sm leading-6 text-zinc-400">Shop select desk, audio, and lighting products with a premium minimalist style.</p>
                <div className="mt-6"><Btn onClick={() => nav('/store')}>Browse sale</Btn></div>
              </div>
              <div className="grid gap-3">
                {sale.map(p => (
                  <button key={p.id} onClick={() => setPick(p)} className="rounded-3xl border border-white/10 bg-white/5 p-3 text-left hover:bg-white/10">
                    <div className="flex items-center gap-3">
                      <img src={p.img} className="h-16 w-16 rounded-2xl object-cover" />
                      <div>
                        <div className="font-medium">{p.name}</div>
                        <div className="text-sm text-zinc-400">{fmt(p.price)}</div>
                      </div>
                    </div>
                  </button>
                ))}
              </div>
            </div>
          </Glass>
          <div className="grid gap-4">
            <Glass className="p-6">
              <div className="text-sm text-zinc-400">Shipping promise</div>
              <div className="mt-2 text-2xl font-semibold">Fast, minimal, transparent</div>
              <p className="mt-2 text-sm leading-6 text-zinc-400">Free shipping on orders over $200, clean order review, and premium confirmation states.</p>
            </Glass>
            <Glass className="p-6">
              <div className="text-sm text-zinc-400">Customer signal</div>
              <div className="mt-2 text-2xl font-semibold">Loved for clean design</div>
              <p className="mt-2 text-sm leading-6 text-zinc-400">Designed to feel calm, modern, and trustworthy across mobile, tablet, and desktop.</p>
            </Glass>
          </div>
        </div>
      </section>
    </div>
  )
}

function ProdCard({ p, add, setPick }: { p: P; add: (p: P, n?: number) => void; setPick: (p: P) => void }) {
  return (
    <motion.div whileHover={{ y: -4 }} className="group">
      <Glass className="overflow-hidden">
        <button onClick={() => setPick(p)} className="block w-full text-left">
          <div className="relative aspect-[4/3] overflow-hidden">
            <img src={p.img} className="h-full w-full object-cover transition duration-500 group-hover:scale-105" />
            <div className="absolute inset-0 bg-gradient-to-t from-black/50 via-transparent to-transparent" />
            <div className="absolute left-3 top-3 flex gap-2">
              {p.sale && <Badge tone="y">Sale</Badge>}
              {p.stock < 10 && <Badge tone="r">Low stock</Badge>}
            </div>
          </div>
        </button>
        <div className="p-4">
          <div className="flex items-start justify-between gap-3">
            <div>
              <button onClick={() => setPick(p)} className="text-left text-lg font-medium transition hover:text-white/80">{p.name}</button>
              <div className="mt-1 text-sm text-zinc-400">{p.sub}</div>
            </div>
            <div className="rounded-2xl border border-white/10 bg-white/5 px-3 py-2 text-sm">{fmt(p.price)}</div>
          </div>
          <div className="mt-4 flex items-center justify-between">
            <div className="flex items-center gap-1 text-sm text-zinc-400"><Star className="h-4 w-4 fill-current" /> {p.rate}</div>
            <Btn className="px-3 py-2" onClick={() => add(p)}>Add</Btn>
          </div>
        </div>
      </Glass>
    </motion.div>
  )
}

function Store({ prods, add, setPick, nav }: { prods: P[]; add: (p: P, n?: number) => void; setPick: (p: P) => void; nav: (p: string) => void }) {
  const [q, setQ] = useState('')
  const [cat, setCat] = useState('All')
  const [sale, setSale] = useState(false)
  const [inst, setInst] = useState(false)
  const [sort, setSort] = useState('feat')

  const rows = useMemo(() => {
    let r = [...prods]
    if (q) r = r.filter(x => `${x.name} ${x.sub} ${x.cat} ${x.tags.join(' ')}`.toLowerCase().includes(q.toLowerCase()))
    if (cat !== 'All') r = r.filter(x => x.cat === cat)
    if (sale) r = r.filter(x => x.sale)
    if (inst) r = r.filter(x => x.stock > 0)
    if (sort === 'pl') r.sort((a, b) => a.price - b.price)
    if (sort === 'ph') r.sort((a, b) => b.price - a.price)
    if (sort === 'az') r.sort((a, b) => a.name.localeCompare(b.name))
    if (sort === 'rt') r.sort((a, b) => b.rate - a.rate)
    if (sort === 'feat') r.sort((a, b) => Number(b.feat) - Number(a.feat) || Number(b.sale) - Number(a.sale))
    return r
  }, [prods, q, cat, sale, inst, sort])

  return (
    <div className="space-y-5">
      <Crumbs nav={nav} items={[{ t: 'Home', k: '/home' }, { t: 'Store' }]} />
      <div className="grid gap-6 lg:grid-cols-[280px_1fr]">
        <Glass className="h-fit p-4 sm:p-5 lg:sticky lg:top-24">
          <Sec eye="Filters" title="Refine" sub="Search, sort, and shape the collection." />
          <div className="space-y-4">
            <Field>
              <Search className="h-4 w-4 text-zinc-500" />
              <input value={q} onChange={e => setQ(e.target.value)} placeholder="Search products" className="w-full bg-transparent text-sm outline-none placeholder:text-zinc-500" />
            </Field>
            <Select v={cat} setV={setCat} opts={['All', ...cats]} />
            <Select v={sort} setV={setSort} opts={[['feat', 'Featured'], ['pl', 'Price low'], ['ph', 'Price high'], ['az', 'Name'], ['rt', 'Top rated']]} />
            <div className="grid gap-3">
              <Check on={sale} setOn={setSale} t="On sale" />
              <Check on={inst} setOn={setInst} t="In stock" />
            </div>
          </div>
        </Glass>
        <div>
          <Sec eye="Store" title="Modern product catalog" sub={`${rows.length} items shown with live mocked interactions.`} right={<Badge tone="b">Responsive grid</Badge>} />
          <div className="mb-4 flex flex-wrap gap-2">
            {cat !== 'All' && <Chip t={cat} onClick={() => setCat('All')} />}
            {sale && <Chip t="Sale" onClick={() => setSale(false)} />}
            {inst && <Chip t="In stock" onClick={() => setInst(false)} />}
            {q && <Chip t={`“${q}”`} onClick={() => setQ('')} />}
          </div>
          {rows.length ? (
            <div className="grid gap-4 sm:grid-cols-2 xl:grid-cols-3">
              {rows.map(p => <ProdCard key={p.id} p={p} add={add} setPick={setPick} />)}
            </div>
          ) : (
            <EmptyStore clear={() => { setQ(''); setCat('All'); setSale(false); setInst(false) }} />
          )}
        </div>
      </div>
    </div>
  )
}

function Item({ p, add, prods, nav, setPick }: { p: P; add: (p: P, n?: number) => void; prods: P[]; nav: (p: string) => void; setPick: (p: P) => void }) {
  const [img, setImg] = useState(p.gallery[0] || p.img)
  const [n, setN] = useState(1)
  const rel = prods.filter(x => x.id !== p.id && x.cat === p.cat).slice(0, 3)

  useEffect(() => {
    setImg(p.gallery[0] || p.img)
    setN(1)
  }, [p])

  return (
    <div className="space-y-8">
      <Crumbs nav={nav} items={[{ t: 'Home', k: '/home' }, { t: 'Store', k: '/store' }, { t: p.name }]} />
      <button onClick={() => nav('/store')} className="inline-flex items-center gap-2 text-sm text-zinc-400 hover:text-white"><ChevronLeft className="h-4 w-4" /> Back to store</button>
      <div className="grid gap-6 lg:grid-cols-[1.1fr_.9fr]">
        <Glass className="p-3 sm:p-4">
          <div className="aspect-[4/3] overflow-hidden rounded-[24px] border border-white/10 bg-black/20">
            <img src={img} className="h-full w-full object-cover" />
          </div>
          <div className="mt-3 grid grid-cols-4 gap-3">
            {(p.gallery.length ? p.gallery : [p.img]).map((g, i) => (
              <button key={i} onClick={() => setImg(g)} className={cls('overflow-hidden rounded-2xl border', img === g ? 'border-white/30' : 'border-white/10')}>
                <img src={g} className="aspect-square h-full w-full object-cover" />
              </button>
            ))}
          </div>
        </Glass>
        <Glass className="p-6 sm:p-7">
          <div className="flex flex-wrap gap-2">
            {p.sale && <Badge tone="y">Sale</Badge>}
            <Badge tone={p.stock < 10 ? 'r' : 'g'}>{p.stock < 10 ? 'Low stock' : 'In stock'}</Badge>
          </div>
          <h1 className="mt-4 text-3xl font-semibold tracking-tight sm:text-4xl">{p.name}</h1>
          <p className="mt-2 text-base text-zinc-400">{p.sub}</p>
          <div className="mt-5 flex items-end gap-3">
            <div className="text-3xl font-semibold">{fmt(p.price)}</div>
            <div className="pb-1 text-zinc-500 line-through">{fmt(p.cmp)}</div>
          </div>
          <p className="mt-5 max-w-xl text-sm leading-7 text-zinc-300">{p.desc}</p>
          <div className="mt-6 flex flex-wrap gap-2">{p.tags.map(t => <Badge key={t}>{t}</Badge>)}</div>
          <div className="mt-8 flex flex-wrap items-center gap-3">
            <Qty n={n} setN={setN} />
            <Btn onClick={() => add(p, n)}>Add to cart</Btn>
            <Btn kind="b" onClick={() => { add(p, n); nav('/checkout') }}>Buy now</Btn>
          </div>
          <div className="mt-8 grid gap-3 sm:grid-cols-3">
            {[[`${p.stock}`, 'Stock'], [p.cat, 'Category'], [`${p.rate} ★`, 'Rating']].map(x => (
              <div key={x[1]} className="rounded-2xl border border-white/10 bg-black/20 p-4">
                <div className="text-lg font-medium">{x[0]}</div>
                <div className="mt-1 text-sm text-zinc-500">{x[1]}</div>
              </div>
            ))}
          </div>
        </Glass>
      </div>
      <section>
        <Sec eye="Related" title="You may also like" sub="More in the same category." />
        <div className="grid gap-4 sm:grid-cols-2 xl:grid-cols-3">{rel.map(x => <ProdCard key={x.id} p={x} add={add} setPick={setPick} />)}</div>
      </section>
    </div>
  )
}

function Checkout({ cart, sub, tax, ship, tot, nav, setCart }: { cart: C[]; sub: number; tax: number; ship: number; tot: number; nav: (p: string) => void; setCart: React.Dispatch<React.SetStateAction<C[]>> }) {
  const [done, setDone] = useState(false)
  const [f, setF] = useState<F>({ n: '', e: '', a: '', c: '', s: '', z: '', card: '', exp: '', cvv: '' })
  const ok = !!(f.n && f.e && f.a && f.c && f.s && f.z && f.card && f.exp && f.cvv && cart.length)

  if (done) {
    return (
      <div className="space-y-5">
        <Crumbs nav={nav} items={[{ t: 'Home', k: '/home' }, { t: 'Store', k: '/store' }, { t: 'Checkout' }]} />
        <Glass className="mx-auto max-w-2xl p-8 text-center">
          <Badge tone="g">Order placed</Badge>
          <h2 className="mt-4 text-3xl font-semibold">Purchase complete</h2>
          <p className="mt-3 text-zinc-400">Your mock order has been created successfully. A simulated confirmation has been generated for this design flow.</p>
          <div className="mt-8 flex justify-center gap-3">
            <Btn onClick={() => nav('/store')}>Continue shopping</Btn>
            <Btn kind="b" onClick={() => nav('/home')}>Back home</Btn>
          </div>
        </Glass>
      </div>
    )
  }

  return (
    <div className="space-y-5">
      <Crumbs nav={nav} items={[{ t: 'Home', k: '/home' }, { t: 'Store', k: '/store' }, { t: 'Checkout' }]} />
      <div className="grid gap-6 lg:grid-cols-[1fr_380px]">
        <Glass className="p-6 sm:p-7">
          <Sec eye="Checkout" title="Complete your order" sub="Mocked billing, shipping, and payment forms." />
          <div className="grid gap-4 sm:grid-cols-2">
            <Input t="Full name" v={f.n} onChange={v => setF({ ...f, n: v })} />
            <Input t="Email" v={f.e} onChange={v => setF({ ...f, e: v })} />
            <Input t="Address" v={f.a} onChange={v => setF({ ...f, a: v })} c="sm:col-span-2" />
            <Input t="City" v={f.c} onChange={v => setF({ ...f, c: v })} />
            <Input t="State" v={f.s} onChange={v => setF({ ...f, s: v })} />
            <Input t="ZIP" v={f.z} onChange={v => setF({ ...f, z: v })} />
            <div className="sm:col-span-2 mt-2 border-t border-white/10 pt-4 text-sm text-zinc-400">Payment</div>
            <Input t="Card number" v={f.card} onChange={v => setF({ ...f, card: v })} c="sm:col-span-2" />
            <Input t="Expiry" v={f.exp} onChange={v => setF({ ...f, exp: v })} />
            <Input t="CVV" v={f.cvv} onChange={v => setF({ ...f, cvv: v })} />
          </div>
          <div className="mt-6 flex flex-wrap gap-3">
            <Btn onClick={() => { if (!ok) return; setDone(true); setCart([]) }}>Place order</Btn>
            <Btn kind="b" onClick={() => nav('/store')}>Back to store</Btn>
          </div>
        </Glass>
        <Glass className="h-fit p-6 lg:sticky lg:top-24">
          <Sec eye="Summary" title="Order review" />
          <div className="space-y-3">
            {cart.map(x => (
              <div key={x.id} className="flex items-center gap-3 rounded-2xl border border-white/10 bg-black/20 p-3">
                <img src={x.img} className="h-14 w-14 rounded-2xl object-cover" />
                <div className="min-w-0 flex-1">
                  <div className="truncate font-medium">{x.name}</div>
                  <div className="text-sm text-zinc-500">Qty {x.qty}</div>
                </div>
                <div>{fmt(x.price * x.qty)}</div>
              </div>
            ))}
          </div>
          <div className="mt-5 space-y-2 border-t border-white/10 pt-4 text-sm">
            <Row a="Subtotal" b={fmt(sub)} />
            <Row a="Shipping" b={ship ? fmt(ship) : 'Free'} />
            <Row a="Tax" b={fmt(tax)} />
            <Row a="Total" b={fmt(tot)} strong />
          </div>
        </Glass>
      </div>
    </div>
  )
}

function Cart({ open, setOpen, cart, qty, rm, sub, tax, ship, tot, nav }: { open: boolean; setOpen: React.Dispatch<React.SetStateAction<boolean>>; cart: C[]; qty: (id: number, d: number) => void; rm: (id: number) => void; sub: number; tax: number; ship: number; tot: number; nav: (p: string) => void }) {
  return (
    <AnimatePresence>
      {open && (
        <>
          <motion.div initial={{ opacity: 0 }} animate={{ opacity: 1 }} exit={{ opacity: 0 }} onClick={() => setOpen(false)} className="fixed inset-0 z-50 bg-black/50 backdrop-blur-sm" />
          <motion.aside initial={{ x: 420 }} animate={{ x: 0 }} exit={{ x: 420 }} className="fixed right-0 top-0 z-[60] flex h-full w-full max-w-md flex-col border-l border-white/10 bg-zinc-950/90 backdrop-blur-2xl">
            <div className="flex items-center justify-between border-b border-white/10 px-5 py-4">
              <div><div className="text-sm text-zinc-400">Cart</div><div className="text-lg font-semibold">Your items</div></div>
              <button onClick={() => setOpen(false)} className="rounded-2xl border border-white/10 bg-white/5 p-2"><X className="h-5 w-5" /></button>
            </div>
            <div className="flex-1 space-y-3 overflow-auto p-5">
              {cart.length ? cart.map(x => (
                <div key={x.id} className="rounded-3xl border border-white/10 bg-white/5 p-3">
                  <div className="flex gap-3">
                    <img src={x.img} className="h-20 w-20 rounded-2xl object-cover" />
                    <div className="min-w-0 flex-1">
                      <div className="font-medium">{x.name}</div>
                      <div className="mt-1 text-sm text-zinc-500">{fmt(x.price)}</div>
                      <div className="mt-3 flex items-center justify-between">
                        <Qty n={x.qty} setN={v => qty(x.id, v - x.qty)} />
                        <button onClick={() => rm(x.id)} className="text-sm text-zinc-500 hover:text-white">Remove</button>
                      </div>
                    </div>
                  </div>
                </div>
              )) : <div className="rounded-3xl border border-dashed border-white/10 p-8 text-center text-sm text-zinc-500">Your cart is empty.</div>}
            </div>
            <div className="border-t border-white/10 p-5">
              <div className="space-y-2 text-sm">
                <Row a="Subtotal" b={fmt(sub)} />
                <Row a="Shipping" b={ship ? fmt(ship) : 'Free'} />
                <Row a="Tax" b={fmt(tax)} />
                <Row a="Total" b={fmt(tot)} strong />
              </div>
              <div className="mt-4 grid gap-3 sm:grid-cols-2">
                <Btn kind="b" onClick={() => setOpen(false)}>Keep shopping</Btn>
                <Btn onClick={() => { setOpen(false); nav('/checkout') }}>Checkout</Btn>
              </div>
            </div>
          </motion.aside>
        </>
      )}
    </AnimatePresence>
  )
}

function Toast({ note, setNote }: { note: string; setNote: React.Dispatch<React.SetStateAction<string>> }) {
  useEffect(() => {
    if (!note) return
    const t = setTimeout(() => setNote(''), 1600)
    return () => clearTimeout(t)
  }, [note, setNote])

  return (
    <AnimatePresence>
      {note && <motion.div initial={{ opacity: 0, y: 12 }} animate={{ opacity: 1, y: 0 }} exit={{ opacity: 0, y: 12 }} className="fixed bottom-4 left-1/2 z-[70] -translate-x-1/2 rounded-2xl border border-white/10 bg-zinc-900/90 px-4 py-3 text-sm shadow-2xl backdrop-blur-xl">{note}</motion.div>}
    </AnimatePresence>
  )
}

function EmptyStore({ clear }: { clear: () => void }) {
  return (
    <Glass className="p-10 text-center">
      <div className="text-2xl font-semibold">No products match those filters</div>
      <p className="mt-2 text-sm text-zinc-400">Try clearing one or more filters to broaden the results.</p>
      <div className="mt-6"><Btn onClick={clear}>Clear filters</Btn></div>
    </Glass>
  )
}

function Field({ children }: { children: React.ReactNode }) {
  return <div className="flex items-center gap-3 rounded-2xl border border-white/10 bg-white/5 px-4 py-3">{children}</div>
}

function Chip({ t, onClick }: { t: string; onClick: () => void }) {
  return <button onClick={onClick} className="inline-flex items-center gap-2 rounded-full border border-white/10 bg-white/5 px-3 py-2 text-sm text-zinc-300 hover:bg-white/10">{t}<X className="h-3 w-3" /></button>
}

function Check({ on, setOn, t }: { on: boolean; setOn: React.Dispatch<React.SetStateAction<boolean>>; t: string }) {
  return <button onClick={() => setOn(!on)} className={cls('flex items-center justify-between rounded-2xl border px-4 py-3 text-sm', on ? 'border-white/20 bg-white/10' : 'border-white/10 bg-white/5')}><span>{t}</span><span className={cls('h-5 w-9 rounded-full p-0.5 transition', on ? 'bg-white' : 'bg-zinc-700')}><span className={cls('block h-4 w-4 rounded-full transition', on ? 'translate-x-4 bg-zinc-900' : 'translate-x-0 bg-white')} /></span></button>
}

function Select({ v, setV, opts }: { v: string; setV: React.Dispatch<React.SetStateAction<string>>; opts: (string | [string, string])[] }) {
  const rows = opts.map(x => Array.isArray(x) ? x : [x, x])
  return <div className="relative"><select value={v} onChange={e => setV(e.target.value)} className="w-full appearance-none rounded-2xl border border-white/10 bg-white/5 px-4 py-3 text-sm outline-none focus:border-white/20">{rows.map(x => <option key={x[0]} value={x[0]} className="bg-zinc-950">{x[1]}</option>)}</select><ChevronDown className="pointer-events-none absolute right-4 top-1/2 h-4 w-4 -translate-y-1/2 text-zinc-500" /></div>
}

function Lab({ t }: { t: string }) {
  return <label className="mb-2 block text-sm text-zinc-400">{t}</label>
}

function Input({ t, v, onChange, c = '' }: { t: string; v: string; onChange: (v: string) => void; c?: string }) {
  return <div className={c}><Lab t={t} /><input value={v} onChange={e => onChange(e.target.value)} className="w-full rounded-2xl border border-white/10 bg-white/5 px-4 py-3 text-sm outline-none focus:border-white/20" /></div>
}

function Row({ a, b, strong }: { a: string; b: string; strong?: boolean }) {
  return <div className={cls('flex items-center justify-between', strong && 'pt-2 text-base font-semibold')}><span className={strong ? 'text-white' : 'text-zinc-400'}>{a}</span><span>{b}</span></div>
}

function Qty({ n, setN }: { n: number; setN: (n: number) => void }) {
  return (
    <div className="flex items-center gap-2 rounded-2xl border border-white/10 bg-white/5 p-2">
      <button onClick={() => setN(Math.max(1, n - 1))} className="rounded-xl px-3 py-2 hover:bg-white/10">-</button>
      <div className="min-w-10 text-center text-sm">{n}</div>
      <button onClick={() => setN(n + 1)} className="rounded-xl px-3 py-2 hover:bg-white/10">+</button>
    </div>
  )
}

