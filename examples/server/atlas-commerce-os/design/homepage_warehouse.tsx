// @ts-nocheck
import React, { useMemo, useState } from 'react'
import { AnimatePresence, motion } from 'framer-motion'
import { ArrowRight, Check, ChevronDown, ChevronRight, Menu, Package, Pencil, Plus, Search, Trash2, Truck, Warehouse, X } from 'lucide-react'

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
  th: number
  sku: string
  feat: boolean
  sale: boolean
  st: 'active' | 'draft' | 'archived'
  rate: number
  img: string
  gallery: string[]
  desc: string
  tags: string[]
  sup: string
}

type Crumb = { t: string; k?: string }
type Act = { id: number; t: string; s: string; at: string }
type Po = { id: number; sup: string; items: { id: number; name: string; qty: number; sku: string }[]; eta: string; st: 'draft' | 'sent' | 'received' }

type Form = {
  id?: number
  slug: string
  name: string
  sub: string
  cat: string
  price: string
  cmp: string
  stock: string
  th: string
  sku: string
  feat: boolean
  sale: boolean
  st: 'active' | 'draft' | 'archived'
  img: string
  desc: string
  tags: string
  sup: string
}

const cats = ['Audio', 'Desk', 'Lighting', 'Travel', 'Wearables']
const sups = ['Nova Supply', 'Luma Works', 'Field Goods', 'Core Time']

const prods0: P[] = [
  { id: 1, slug: 'aero-headphones', name: 'Aero Headphones', sub: 'Wireless spatial audio', cat: 'Audio', price: 249, cmp: 319, stock: 18, th: 8, sku: 'AUD-101', feat: true, sale: true, st: 'active', rate: 4.8, img: 'https://images.unsplash.com/photo-1505740420928-5e560c06d30e?auto=format&fit=crop&w=1200&q=80', gallery: ['https://images.unsplash.com/photo-1505740420928-5e560c06d30e?auto=format&fit=crop&w=1200&q=80'], desc: 'Clean sound, soft pads, long battery life.', tags: ['wireless', 'premium'], sup: 'Nova Supply' },
  { id: 2, slug: 'halo-lamp', name: 'Halo Lamp', sub: 'Ambient desk glow', cat: 'Lighting', price: 129, cmp: 159, stock: 9, th: 10, sku: 'LGT-204', feat: true, sale: false, st: 'active', rate: 4.6, img: 'https://images.unsplash.com/photo-1507473885765-e6ed057f782c?auto=format&fit=crop&w=1200&q=80', gallery: ['https://images.unsplash.com/photo-1507473885765-e6ed057f782c?auto=format&fit=crop&w=1200&q=80'], desc: 'Soft light with a sculpted profile for modern desks.', tags: ['ambient', 'desk'], sup: 'Luma Works' },
  { id: 3, slug: 'atlas-pack', name: 'Atlas Pack', sub: 'Minimal travel bag', cat: 'Travel', price: 189, cmp: 229, stock: 31, th: 12, sku: 'TRV-310', feat: false, sale: true, st: 'active', rate: 4.7, img: 'https://images.unsplash.com/photo-1542291026-7eec264c27ff?auto=format&fit=crop&w=1200&q=80', gallery: ['https://images.unsplash.com/photo-1542291026-7eec264c27ff?auto=format&fit=crop&w=1200&q=80'], desc: 'Balanced storage with premium fabric and clean lines.', tags: ['travel', 'carry'], sup: 'Field Goods' },
  { id: 4, slug: 'drift-keyboard', name: 'Drift Keyboard', sub: 'Low profile mechanical feel', cat: 'Desk', price: 139, cmp: 169, stock: 24, th: 10, sku: 'DSK-402', feat: true, sale: false, st: 'active', rate: 4.5, img: 'https://images.unsplash.com/photo-1511467687858-23d96c32e4ae?auto=format&fit=crop&w=1200&q=80', gallery: ['https://images.unsplash.com/photo-1511467687858-23d96c32e4ae?auto=format&fit=crop&w=1200&q=80'], desc: 'Quiet switches and compact layout for focused work.', tags: ['desk', 'input'], sup: 'Nova Supply' },
  { id: 5, slug: 'pulse-watch', name: 'Pulse Watch', sub: 'Everyday motion tracking', cat: 'Wearables', price: 279, cmp: 329, stock: 6, th: 10, sku: 'WRB-509', feat: true, sale: true, st: 'active', rate: 4.9, img: 'https://images.unsplash.com/photo-1523275335684-37898b6baf30?auto=format&fit=crop&w=1200&q=80', gallery: ['https://images.unsplash.com/photo-1523275335684-37898b6baf30?auto=format&fit=crop&w=1200&q=80'], desc: 'Slim wearable with clear metrics and refined materials.', tags: ['fitness', 'smart'], sup: 'Core Time' },
  { id: 6, slug: 'echo-speaker', name: 'Echo Speaker', sub: 'Compact room-filling sound', cat: 'Audio', price: 159, cmp: 199, stock: 14, th: 6, sku: 'AUD-118', feat: false, sale: false, st: 'draft', rate: 4.4, img: 'https://images.unsplash.com/photo-1545454675-3531b543be5d?auto=format&fit=crop&w=1200&q=80', gallery: ['https://images.unsplash.com/photo-1545454675-3531b543be5d?auto=format&fit=crop&w=1200&q=80'], desc: 'Simple form, balanced acoustics, and subtle controls.', tags: ['speaker', 'home'], sup: 'Luma Works' },
  { id: 7, slug: 'glide-stand', name: 'Glide Stand', sub: 'Elevated laptop stand', cat: 'Desk', price: 89, cmp: 109, stock: 42, th: 16, sku: 'DSK-455', feat: false, sale: true, st: 'active', rate: 4.3, img: 'https://images.unsplash.com/photo-1516321318423-f06f85e504b3?auto=format&fit=crop&w=1200&q=80', gallery: ['https://images.unsplash.com/photo-1516321318423-f06f85e504b3?auto=format&fit=crop&w=1200&q=80'], desc: 'Open aluminum form for airflow and ergonomic lift.', tags: ['desk', 'setup'], sup: 'Field Goods' },
  { id: 8, slug: 'nova-bottle', name: 'Nova Bottle', sub: 'Insulated daily carry', cat: 'Travel', price: 44, cmp: 54, stock: 63, th: 20, sku: 'TRV-325', feat: false, sale: false, st: 'active', rate: 4.2, img: 'https://images.unsplash.com/photo-1602143407151-7111542de6e8?auto=format&fit=crop&w=1200&q=80', gallery: ['https://images.unsplash.com/photo-1602143407151-7111542de6e8?auto=format&fit=crop&w=1200&q=80'], desc: 'Temperature retention and a quiet minimal silhouette.', tags: ['carry', 'daily'], sup: 'Core Time' },
]

const acts0: Act[] = [
  { id: 1, t: 'Inventory sync', s: '14 products updated', at: '8m ago' },
  { id: 2, t: 'Purchase order sent', s: 'Nova Supply · 3 items', at: '28m ago' },
  { id: 3, t: 'Stock adjusted', s: 'Pulse Watch +12 units', at: '1h ago' },
]

const po0: Po[] = [
  { id: 1, sup: 'Nova Supply', items: [{ id: 1, name: 'Pulse Watch', qty: 24, sku: 'WRB-509' }, { id: 2, name: 'Halo Lamp', qty: 18, sku: 'LGT-204' }], eta: 'Apr 02', st: 'sent' },
  { id: 2, sup: 'Field Goods', items: [{ id: 3, name: 'Atlas Pack', qty: 16, sku: 'TRV-310' }], eta: 'Apr 05', st: 'draft' },
]

export default function App() {
  const [route, setRoute] = useState('/home')
  const [menu, setMenu] = useState(false)
  const [prods, setProds] = useState<P[]>(prods0)
  const [acts, setActs] = useState<Act[]>(acts0)
  const [pos, setPos] = useState<Po[]>(po0)
  const [note, setNote] = useState('')
  const [edit, setEdit] = useState<P | null>(null)

  const nav = (p: string) => {
    setRoute(p)
    setMenu(false)
    if (typeof window !== 'undefined') window.scrollTo({ top: 0, behavior: 'smooth' })
  }

  const low = prods.filter(x => x.stock <= x.th)
  const val = prods.reduce((a, b) => a + b.stock * b.price, 0)

  const save = (f: Form) => {
    const p: P = {
      id: f.id || Date.now(),
      slug: f.slug,
      name: f.name,
      sub: f.sub,
      cat: f.cat,
      price: Number(f.price) || 0,
      cmp: Number(f.cmp) || 0,
      stock: Number(f.stock) || 0,
      th: Number(f.th) || 0,
      sku: f.sku,
      feat: f.feat,
      sale: f.sale,
      st: f.st,
      rate: 4.5,
      img: f.img,
      gallery: [f.img],
      desc: f.desc,
      tags: f.tags.split(',').map(x => x.trim()).filter(Boolean),
      sup: f.sup,
    }
    setProds(s => f.id ? s.map(x => x.id === f.id ? p : x) : [p, ...s])
    setActs(a => [{ id: Date.now(), t: f.id ? 'Item updated' : 'Item created', s: p.name, at: 'Now' }, ...a])
    setEdit(null)
    setNote(f.id ? `${p.name} updated` : `${p.name} created`)
    nav('/warehouse/items')
  }

  const del = (id: number) => {
    const p = prods.find(x => x.id === id)
    setProds(s => s.filter(x => x.id !== id))
    setActs(a => [{ id: Date.now(), t: 'Item removed', s: p?.name || 'Unknown item', at: 'Now' }, ...a])
    setNote(`${p?.name || 'Item'} removed`)
  }

  const mkPo = (sup: string, rows: { id: number; name: string; qty: number; sku: string }[]) => {
    if (!rows.length) return
    const po: Po = { id: Date.now(), sup, items: rows, eta: 'Apr 09', st: 'sent' }
    setPos(s => [po, ...s])
    setActs(a => [{ id: Date.now(), t: 'Purchase order sent', s: `${sup} · ${rows.length} items`, at: 'Now' }, ...a])
    setNote(`PO sent to ${sup}`)
    nav('/warehouse/orders')
  }

  return (
    <div className="min-h-screen bg-zinc-950 text-zinc-100">
      <style>{`button, select, [role="button"] { cursor: pointer; } input, textarea { cursor: text; } button:disabled, select:disabled, input:disabled, textarea:disabled { cursor: not-allowed; }`}</style>
      <Bg />
      <Top nav={nav} route={route} menu={menu} setMenu={setMenu} />
      <main className="mx-auto max-w-7xl px-4 pb-16 pt-24 sm:px-6 lg:px-8">
        {route === '/home' && <Home nav={nav} prods={prods} low={low.length} val={val} />}
        {route === '/warehouse' && <Dash nav={nav} prods={prods} acts={acts} pos={pos} low={low} val={val} />}
        {route === '/warehouse/items' && <Items nav={nav} prods={prods} setEdit={setEdit} del={del} />}
        {route === '/warehouse/new' && <ItemForm nav={nav} save={save} item={edit} />}
        {route === '/warehouse/orders' && <Orders nav={nav} pos={pos} />}
        {route === '/warehouse/reorder' && <Reorder nav={nav} low={low} mkPo={mkPo} />}
      </main>
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

function Top({ nav, route, menu, setMenu }: { nav: (p: string) => void; route: string; menu: boolean; setMenu: React.Dispatch<React.SetStateAction<boolean>> }) {
  const links = [
    { k: '/home', t: 'Home' },
    { k: '/warehouse', t: 'Warehouse' },
    { k: '/warehouse/items', t: 'Items' },
    { k: '/warehouse/orders', t: 'Orders' },
  ]
  return (
    <>
      <div className="fixed inset-x-0 top-0 z-50 border-b border-white/10 bg-white/5 backdrop-blur-xl">
        <div className="mx-auto flex max-w-7xl items-center justify-between px-4 py-3 sm:px-6 lg:px-8">
          <button onClick={() => nav('/home')} className="flex items-center gap-3">
            <div className="grid h-10 w-10 place-items-center rounded-2xl border border-white/15 bg-white/10 shadow-2xl shadow-black/30"><Warehouse className="h-5 w-5" /></div>
            <div className="text-left"><div className="text-sm text-zinc-100/90">Modern Ops</div><div className="text-base font-semibold text-white">Atelier Warehouse</div></div>
          </button>
          <div className="hidden items-center gap-2 md:flex">
            {links.map(x => <button key={x.k} onClick={() => nav(x.k)} className={cls('rounded-2xl px-4 py-2 text-sm transition', route === x.k ? 'border border-white/15 bg-white/10 shadow-lg shadow-black/20' : 'text-zinc-100/90 hover:bg-white/5 hover:text-white')}>{x.t}</button>)}
          </div>
          <button onClick={() => setMenu(v => !v)} className="rounded-2xl border border-white/15 bg-white/10 p-3 md:hidden"><Menu className="h-5 w-5" /></button>
        </div>
      </div>
      <AnimatePresence>
        {menu && <motion.div initial={{ opacity: 0, y: -8 }} animate={{ opacity: 1, y: 0 }} exit={{ opacity: 0, y: -8 }} className="fixed inset-x-4 top-20 z-40 rounded-3xl border border-white/10 bg-zinc-900/80 p-3 backdrop-blur-2xl md:hidden">{links.map(x => <button key={x.k} onClick={() => nav(x.k)} className="flex w-full items-center justify-between rounded-2xl px-4 py-3 text-left text-zinc-200 hover:bg-white/5">{x.t}<ArrowRight className="h-4 w-4" /></button>)}</motion.div>}
      </AnimatePresence>
    </>
  )
}

function Glass({ className = '', children }: { className?: string; children: React.ReactNode }) {
  return <div className={cls('rounded-[28px] border border-white/10 bg-white/5 backdrop-blur-xl shadow-2xl shadow-black/20', className)}>{children}</div>
}

function Btn({ className = '', onClick, children, kind = 'a', type = 'button' }: { className?: string; onClick?: () => void; children: React.ReactNode; kind?: 'a' | 'b'; type?: 'button' | 'submit' }) {
  return <button type={type} onClick={onClick} className={cls('rounded-2xl px-4 py-3 text-sm font-semibold transition active:scale-[0.98]', kind === 'a' ? 'bg-white text-zinc-950 shadow-xl shadow-black/20 hover:bg-zinc-200' : 'border border-white/25 bg-white/16 text-white shadow-lg shadow-black/20 hover:bg-white/22', className)}>{children}</button>
}

function Sec({ eye, title, sub, right }: { eye: string; title: string; sub?: string; right?: React.ReactNode }) {
  return <div className="mb-6 flex items-end justify-between gap-4"><div><div className="mb-2 text-xs font-semibold uppercase tracking-[0.24em] text-white/75">{eye}</div><h2 className="text-3xl font-semibold tracking-tight text-white sm:text-4xl">{title}</h2>{sub && <p className="mt-3 max-w-2xl text-base leading-7 text-white/82">{sub}</p>}</div>{right}</div>
}

function Badge({ children, tone = 'z' }: { children: React.ReactNode; tone?: 'z' | 'g' | 'b' | 'y' | 'r' }) {
  const m = { z: 'border-white/10 bg-white/5 text-zinc-100/90', g: 'border-emerald-400/20 bg-emerald-400/10 text-emerald-200', b: 'border-cyan-400/20 bg-cyan-400/10 text-cyan-200', y: 'border-amber-400/20 bg-amber-400/10 text-amber-200', r: 'border-rose-400/20 bg-rose-400/10 text-rose-200' }
  return <span className={cls('inline-flex items-center rounded-full border px-3 py-1 text-xs', m[tone])}>{children}</span>
}

function Crumbs({ nav, items }: { nav: (p: string) => void; items: Crumb[] }) {
  return <div className="mb-6 flex flex-wrap items-center gap-2 text-sm text-white/70">{items.map((x, i) => <React.Fragment key={`${x.t}-${i}`}>{i > 0 && <ChevronRight className="h-3.5 w-3.5 text-white/45" />}{x.k ? <button onClick={() => nav(x.k!)} className="transition hover:text-white">{x.t}</button> : <span className="font-semibold text-white">{x.t}</span>}</React.Fragment>)}</div>
}

function Home({ nav, prods, low, val }: { nav: (p: string) => void; prods: P[]; low: number; val: number }) {
  return (
    <div className="space-y-10">
      <Glass className="relative overflow-hidden p-6 sm:p-8 lg:p-10">
        <div className="absolute inset-y-0 right-0 hidden w-1/2 bg-gradient-to-l from-white/10 to-transparent lg:block" />
        <div className="grid items-center gap-8 lg:grid-cols-2">
          <div>
            <Badge tone="b">Warehouse control center</Badge>
            <h1 className="mt-4 text-4xl font-semibold tracking-tight sm:text-5xl lg:text-6xl">Run inventory, purchase orders, and product records from one polished workspace.</h1>
            <p className="mt-4 max-w-xl text-base leading-7 text-zinc-100/90 sm:text-lg">A full warehouse mock with inventory CRUD, low-stock signals, reorder flows, purchase orders, activity tracking, and responsive admin layouts.</p>
            <div className="mt-8 flex flex-wrap gap-3"><Btn onClick={() => nav('/warehouse')}>Open dashboard</Btn><Btn kind="b" onClick={() => nav('/warehouse/items')}>Manage items</Btn></div>
            <div className="mt-8 grid grid-cols-3 gap-3">
              <Glass className="p-4"><div className="text-2xl font-semibold">{prods.length}</div><div className="mt-1 text-sm text-zinc-200">Products</div></Glass>
              <Glass className="p-4"><div className="text-2xl font-semibold">{low}</div><div className="mt-1 text-sm text-zinc-200">Low stock</div></Glass>
              <Glass className="p-4"><div className="text-2xl font-semibold">{fmt(val)}</div><div className="mt-1 text-sm text-zinc-200">Inventory value</div></Glass>
            </div>
          </div>
          <div className="grid gap-4 sm:grid-cols-2">
            <Glass className="p-5 sm:col-span-2"><div className="mb-3 flex items-center justify-between"><Badge tone="g">Modules</Badge><Package className="h-4 w-4 text-zinc-200" /></div><div className="grid gap-3 sm:grid-cols-3">{['Dashboard','Inventory CRUD','Purchase Orders'].map(x => <div key={x} className="rounded-2xl border border-white/10 bg-black/20 p-4 text-sm text-zinc-200">{x}</div>)}</div></Glass>
            <Glass className="p-5"><div className="mb-3 flex items-center justify-between"><Badge tone="y">Alerts</Badge><Truck className="h-4 w-4 text-zinc-200" /></div><div className="text-xl font-medium">Reorder low-stock items</div><p className="mt-2 text-sm leading-6 text-zinc-200">Prebuilt flow to generate supplier orders from shortage thresholds.</p></Glass>
            <Glass className="p-5"><div className="mb-3 flex items-center justify-between"><Badge tone="b">Forms</Badge><Plus className="h-4 w-4 text-zinc-200" /></div><div className="text-xl font-medium">Fast item editing</div><p className="mt-2 text-sm leading-6 text-zinc-200">Compact product forms with preview, tags, supplier, stock, and status.</p></Glass>
          </div>
        </div>
      </Glass>
    </div>
  )
}

function Dash({ nav, prods, acts, pos, low, val }: { nav: (p: string) => void; prods: P[]; acts: Act[]; pos: Po[]; low: P[]; val: number }) {
  return (
    <div className="space-y-6 text-white">
      <Glass className="p-6 sm:p-7">
        <Crumbs nav={nav} items={[{ t: 'Home', k: '/home' }, { t: 'Warehouse' }]} />
        <Sec eye="Overview" title="Warehouse dashboard" sub="A compact operations view with stock health, activity, and purchase order visibility." right={<div className="flex gap-3"><Btn kind="b" className="min-w-24" onClick={() => nav('/warehouse/reorder')}>Reorder</Btn><Btn className="min-w-24" onClick={() => nav('/warehouse/new')}>New item</Btn></div>} />
      </Glass>
      <div className="grid gap-4 sm:grid-cols-2 xl:grid-cols-4">
        <Stat t="Products" v={`${prods.length}`} icon={<Package className="h-5 w-5" />} />
        <Stat t="Low stock" v={`${low.length}`} icon={<Truck className="h-5 w-5" />} tone="y" />
        <Stat t="Value" v={fmt(val)} icon={<Warehouse className="h-5 w-5" />} tone="b" />
        <Stat t="POs" v={`${pos.length}`} icon={<Check className="h-5 w-5" />} tone="g" />
      </div>
      <div className="grid gap-4 xl:grid-cols-[1.1fr_.9fr]">
        <Glass className="p-5">
          <Sec eye="Attention" title="Low stock items" right={<Btn kind="b" onClick={() => nav('/warehouse/reorder')}>Create PO</Btn>} />
          <div className="space-y-3">
            {low.length ? low.map(x => <div key={x.id} className="flex items-center justify-between rounded-2xl border border-white/10 bg-black/20 p-4"><div><div className="font-medium">{x.name}</div><div className="text-sm text-zinc-100/90">{x.sku} · {x.sup}</div></div><Badge tone="y">{x.stock} / {x.th}</Badge></div>) : <Empty t="No low-stock items" s="Everything is currently above threshold." />}
          </div>
        </Glass>
        <div className="grid gap-4">
          <Glass className="p-5">
            <Sec eye="Recent" title="Activity feed" />
            <div className="space-y-3">{acts.map(x => <div key={x.id} className="rounded-2xl border border-white/10 bg-black/20 p-4"><div className="font-medium">{x.t}</div><div className="mt-1 text-sm text-zinc-200">{x.s}</div><div className="mt-2 text-xs text-zinc-100/90">{x.at}</div></div>)}</div>
          </Glass>
          <Glass className="p-5">
            <Sec eye="Orders" title="Purchase orders" right={<Btn kind="b" onClick={() => nav('/warehouse/orders')}>View all</Btn>} />
            <div className="space-y-3">{pos.slice(0, 3).map(x => <div key={x.id} className="rounded-2xl border border-white/10 bg-black/20 p-4"><div className="flex items-center justify-between"><div className="font-medium">{x.sup}</div><Badge tone={x.st === 'sent' ? 'b' : x.st === 'received' ? 'g' : 'z'}>{x.st}</Badge></div><div className="mt-1 text-sm text-zinc-200">{x.items.length} items · ETA {x.eta}</div></div>)}</div>
          </Glass>
        </div>
      </div>
    </div>
  )
}

function Items({ nav, prods, setEdit, del }: { nav: (p: string) => void; prods: P[]; setEdit: React.Dispatch<React.SetStateAction<P | null>>; del: (id: number) => void }) {
  const [q, setQ] = useState('')
  const [cat, setCat] = useState('All')
  const [st, setSt] = useState('All')
  const rows = useMemo(() => prods.filter(x => (cat === 'All' || x.cat === cat) && (st === 'All' || x.st === st) && `${x.name} ${x.sku} ${x.sup}`.toLowerCase().includes(q.toLowerCase())), [prods, q, cat, st])
  return (
    <div className="space-y-5">
      <Glass className="p-6 sm:p-7">
        <Crumbs nav={nav} items={[{ t: 'Home', k: '/home' }, { t: 'Warehouse', k: '/warehouse' }, { t: 'Items' }]} />
        <Sec eye="Inventory" title="Product records" sub="Search, filter, edit, and remove warehouse items." right={<Btn onClick={() => { setEdit(null); nav('/warehouse/new') }}>New item</Btn>} />
      </Glass>
      <Glass className="p-4"><div className="grid gap-3 md:grid-cols-[1fr_220px_220px]"><Field><Search className="h-4 w-4 text-zinc-100/90" /><input value={q} onChange={e => setQ(e.target.value)} placeholder="Search name, sku, supplier" className="w-full bg-transparent text-sm outline-none placeholder:text-zinc-100/90" /></Field><Select v={cat} setV={setCat} opts={['All', ...cats]} /><Select v={st} setV={setSt} opts={['All', 'active', 'draft', 'archived']} /></div></Glass>
      <div className="grid gap-3 xl:hidden">{rows.map(x => <ItemCard key={x.id} x={x} nav={nav} setEdit={setEdit} del={del} />)}</div>
      <Glass className="hidden overflow-hidden xl:block">
        <table className="w-full text-left text-sm">
          <thead className="border-b border-white/10 bg-white/5 text-zinc-200"><tr>{['Item','SKU','Category','Stock','Threshold','Price','Supplier','Status','Actions'].map(h => <th key={h} className="px-4 py-4 font-medium">{h}</th>)}</tr></thead>
          <tbody>{rows.map(x => <tr key={x.id} className="border-b border-white/5 last:border-b-0"><td className="px-4 py-4"><div className="flex items-center gap-3"><img src={x.img} className="h-12 w-12 rounded-2xl object-cover" /><div><div className="font-medium">{x.name}</div><div className="text-zinc-100/90">{x.sub}</div></div></div></td><td className="px-4 py-4 text-zinc-200">{x.sku}</td><td className="px-4 py-4 text-zinc-200">{x.cat}</td><td className="px-4 py-4"><Badge tone={x.stock <= x.th ? 'y' : 'g'}>{x.stock}</Badge></td><td className="px-4 py-4 text-zinc-200">{x.th}</td><td className="px-4 py-4">{fmt(x.price)}</td><td className="px-4 py-4 text-zinc-200">{x.sup}</td><td className="px-4 py-4"><Badge tone={x.st === 'active' ? 'g' : x.st === 'draft' ? 'z' : 'r'}>{x.st}</Badge></td><td className="px-4 py-4"><div className="flex gap-2"><button onClick={() => { setEdit(x); nav('/warehouse/new') }} className="rounded-xl border border-white/10 bg-white/5 p-2 hover:bg-white/10"><Pencil className="h-4 w-4" /></button><button onClick={() => del(x.id)} className="rounded-xl border border-white/10 bg-white/5 p-2 hover:bg-white/10"><Trash2 className="h-4 w-4" /></button></div></td></tr>)}</tbody>
        </table>
      </Glass>
    </div>
  )
}

function ItemCard({ x, nav, setEdit, del }: { x: P; nav: (p: string) => void; setEdit: React.Dispatch<React.SetStateAction<P | null>>; del: (id: number) => void }) {
  return <Glass className="p-4"><div className="flex gap-4"><img src={x.img} className="h-20 w-20 rounded-3xl object-cover" /><div className="min-w-0 flex-1"><div className="flex items-start justify-between gap-3"><div><div className="font-medium">{x.name}</div><div className="text-sm text-zinc-100/90">{x.sku} · {x.sup}</div></div><Badge tone={x.stock <= x.th ? 'y' : 'g'}>{x.stock}</Badge></div><div className="mt-3 flex items-center justify-between"><div className="text-sm text-zinc-200">{fmt(x.price)} · {x.st}</div><div className="flex gap-2"><button onClick={() => { setEdit(x); nav('/warehouse/new') }} className="rounded-xl border border-white/10 bg-white/5 p-2"><Pencil className="h-4 w-4" /></button><button onClick={() => del(x.id)} className="rounded-xl border border-white/10 bg-white/5 p-2"><Trash2 className="h-4 w-4" /></button></div></div></div></div></Glass>
}

function ItemForm({ nav, save, item }: { nav: (p: string) => void; save: (f: Form) => void; item: P | null }) {
  const [f, setF] = useState<Form>(() => item ? { id: item.id, slug: item.slug, name: item.name, sub: item.sub, cat: item.cat, price: String(item.price), cmp: String(item.cmp), stock: String(item.stock), th: String(item.th), sku: item.sku, feat: item.feat, sale: item.sale, st: item.st, img: item.img, desc: item.desc, tags: item.tags.join(', '), sup: item.sup } : { slug: '', name: '', sub: '', cat: cats[0], price: '99', cmp: '129', stock: '10', th: '5', sku: '', feat: false, sale: false, st: 'active', img: 'https://images.unsplash.com/photo-1516321318423-f06f85e504b3?auto=format&fit=crop&w=1200&q=80', desc: '', tags: 'new', sup: sups[0] })
  return (
    <div className="space-y-5">
      <Crumbs nav={nav} items={[{ t: 'Home', k: '/home' }, { t: 'Warehouse', k: '/warehouse' }, { t: 'Items', k: '/warehouse/items' }, { t: item ? 'Edit item' : 'New item' }]} />
      <div className="grid gap-6 lg:grid-cols-[1fr_340px]">
        <Glass className="p-6 sm:p-7">
          <Sec eye="Editor" title={item ? 'Edit product record' : 'Create product record'} sub="Manage product metadata, inventory, supplier, and status." />
          <div className="grid gap-4 sm:grid-cols-2">
            <Input t="Name" v={f.name} onChange={v => setF({ ...f, name: v })} />
            <Input t="Slug" v={f.slug} onChange={v => setF({ ...f, slug: v })} />
            <Input t="Subtitle" v={f.sub} onChange={v => setF({ ...f, sub: v })} c="sm:col-span-2" />
            <div><Lab t="Category" /><Select v={f.cat} setV={v => setF({ ...f, cat: v })} opts={cats} /></div>
            <Input t="SKU" v={f.sku} onChange={v => setF({ ...f, sku: v })} />
            <Input t="Price" v={f.price} onChange={v => setF({ ...f, price: v })} />
            <Input t="Compare" v={f.cmp} onChange={v => setF({ ...f, cmp: v })} />
            <Input t="Stock" v={f.stock} onChange={v => setF({ ...f, stock: v })} />
            <Input t="Threshold" v={f.th} onChange={v => setF({ ...f, th: v })} />
            <div><Lab t="Status" /><Select v={f.st} setV={v => setF({ ...f, st: v as Form['st'] })} opts={['active', 'draft', 'archived']} /></div>
            <div><Lab t="Supplier" /><Select v={f.sup} setV={v => setF({ ...f, sup: v })} opts={sups} /></div>
            <Input t="Image URL" v={f.img} onChange={v => setF({ ...f, img: v })} c="sm:col-span-2" />
            <div className="sm:col-span-2"><Lab t="Description" /><textarea value={f.desc} onChange={e => setF({ ...f, desc: e.target.value })} className="min-h-28 w-full rounded-2xl border border-white/10 bg-white/5 px-4 py-3 text-sm outline-none focus:border-white/20" /></div>
            <Input t="Tags" v={f.tags} onChange={v => setF({ ...f, tags: v })} c="sm:col-span-2" />
            <div className="sm:col-span-2 grid gap-3 sm:grid-cols-2"><CheckRow on={f.feat} setOn={v => setF({ ...f, feat: v })} t="Featured" /><CheckRow on={f.sale} setOn={v => setF({ ...f, sale: v })} t="On sale" /></div>
          </div>
          <div className="mt-6 flex gap-3"><Btn onClick={() => save(f)}>Save item</Btn><Btn kind="b" onClick={() => nav('/warehouse/items')}>Cancel</Btn></div>
        </Glass>
        <Glass className="h-fit p-5 lg:sticky lg:top-24">
          <Sec eye="Preview" title="Item card" />
          <div className="overflow-hidden rounded-[28px] border border-white/10 bg-black/20"><img src={f.img} className="aspect-[4/3] w-full object-cover" /><div className="p-4"><div className="text-lg font-medium">{f.name || 'Untitled item'}</div><div className="mt-1 text-sm text-zinc-200">{f.sub || 'Short product subtitle'}</div><div className="mt-3 flex items-center justify-between"><Badge tone={(Number(f.stock) || 0) <= (Number(f.th) || 0) ? 'y' : 'g'}>{f.stock} stock</Badge><div>{fmt(Number(f.price) || 0)}</div></div></div></div>
        </Glass>
      </div>
    </div>
  )
}

function Orders({ nav, pos }: { nav: (p: string) => void; pos: Po[] }) {
  return (
    <div className="space-y-5">
      <Glass className="p-6 sm:p-7">
        <Crumbs nav={nav} items={[{ t: 'Home', k: '/home' }, { t: 'Warehouse', k: '/warehouse' }, { t: 'Orders' }]} />
        <Sec eye="Purchasing" title="Purchase orders" sub="Track supplier orders and expected arrivals." right={<Btn kind="b" onClick={() => nav('/warehouse/reorder')}>New PO</Btn>} />
      </Glass>
      <div className="grid gap-4">{pos.map(x => <Glass key={x.id} className="p-5"><div className="flex flex-wrap items-start justify-between gap-4"><div><div className="text-xl font-medium">{x.sup}</div><div className="mt-1 text-sm text-zinc-200">ETA {x.eta} · {x.items.length} items</div></div><Badge tone={x.st === 'sent' ? 'b' : x.st === 'received' ? 'g' : 'z'}>{x.st}</Badge></div><div className="mt-4 grid gap-3">{x.items.map(i => <div key={i.id} className="flex items-center justify-between rounded-2xl border border-white/10 bg-black/20 p-4"><div><div className="font-medium">{i.name}</div><div className="text-sm text-zinc-100/90">{i.sku}</div></div><div className="text-sm text-zinc-100/90">Qty {i.qty}</div></div>)}</div></Glass>)}</div>
    </div>
  )
}

function Reorder({ nav, low, mkPo }: { nav: (p: string) => void; low: P[]; mkPo: (sup: string, rows: { id: number; name: string; qty: number; sku: string }[]) => void }) {
  const [sup, setSup] = useState(low[0]?.sup || sups[0])
  const [rows, setRows] = useState(low.map(x => ({ id: x.id, name: x.name, sku: x.sku, qty: Math.max(x.th * 2 - x.stock, 1), sup: x.sup })))
  const shown = rows.filter(x => x.sup === sup)
  const bump = (id: number, d: number) => setRows(s => s.map(x => x.id === id ? { ...x, qty: Math.max(1, x.qty + d) } : x))
  return (
    <div className="space-y-5">
      <Crumbs nav={nav} items={[{ t: 'Home', k: '/home' }, { t: 'Warehouse', k: '/warehouse' }, { t: 'Reorder' }]} />
      <Sec eye="Restock" title="Create supplier order" sub="Generate a purchase order from low-stock items." right={<Btn onClick={() => mkPo(sup, shown.map(({ id, name, qty, sku }) => ({ id, name, qty, sku })))}>Send PO</Btn>} />
      <Glass className="p-4"><div className="grid gap-3 md:grid-cols-[260px_1fr]"><div><Lab t="Supplier" /><Select v={sup} setV={setSup} opts={[...new Set(rows.map(x => x.sup))]} /></div><div className="rounded-2xl border border-white/10 bg-black/20 px-4 py-3 text-sm text-zinc-200">{shown.length} items selected for {sup}</div></div></Glass>
      <div className="grid gap-3">{shown.map(x => <Glass key={x.id} className="p-4"><div className="flex flex-wrap items-center gap-4"><div className="min-w-0 flex-1"><div className="font-medium">{x.name}</div><div className="text-sm text-zinc-100/90">{x.sku}</div></div><div className="flex items-center gap-2 rounded-2xl border border-white/10 bg-white/5 p-2"><button onClick={() => bump(x.id, -1)} className="rounded-xl px-3 py-2 hover:bg-white/10">-</button><div className="min-w-10 text-center text-sm">{x.qty}</div><button onClick={() => bump(x.id, 1)} className="rounded-xl px-3 py-2 hover:bg-white/10">+</button></div><div className="text-sm text-zinc-200">ETA 5–7 days</div></div></Glass>)}{!shown.length && <Empty t="No reorder items for this supplier" s="Switch suppliers or adjust thresholds on products." />}</div>
    </div>
  )
}

function Stat({ t, v, icon, tone = 'z' }: { t: string; v: string; icon: React.ReactNode; tone?: 'z' | 'g' | 'b' | 'y' | 'r' }) {
  const ring = tone === 'b' ? 'from-cyan-400/20' : tone === 'y' ? 'from-amber-400/20' : tone === 'r' ? 'from-rose-400/20' : tone === 'g' ? 'from-emerald-400/20' : 'from-white/10'
  return <Glass className="relative overflow-hidden p-5"><div className={cls('absolute inset-0 bg-gradient-to-br', ring, 'to-transparent')} /><div className="relative flex items-start justify-between gap-3"><div><div className="text-sm font-medium text-zinc-200">{t}</div><div className="mt-2 text-3xl font-semibold tracking-tight text-white">{v}</div></div><div className="grid h-11 w-11 place-items-center rounded-2xl border border-white/10 bg-white/10 text-white">{icon}</div></div></Glass>
}

function Empty({ t, s }: { t: string; s: string }) {
  return <div className="rounded-2xl border border-dashed border-white/10 p-8 text-center"><div className="text-lg font-medium text-white">{t}</div><div className="mt-2 text-sm text-zinc-100/90">{s}</div></div>
}

function Toast({ note, setNote }: { note: string; setNote: React.Dispatch<React.SetStateAction<string>> }) {
  React.useEffect(() => {
    if (!note) return
    const t = setTimeout(() => setNote(''), 1600)
    return () => clearTimeout(t)
  }, [note, setNote])
  return <AnimatePresence>{note && <motion.div initial={{ opacity: 0, y: 12 }} animate={{ opacity: 1, y: 0 }} exit={{ opacity: 0, y: 12 }} className="fixed bottom-4 left-1/2 z-[70] -translate-x-1/2 rounded-2xl border border-white/10 bg-zinc-900/90 px-4 py-3 text-sm shadow-2xl backdrop-blur-xl">{note}</motion.div>}</AnimatePresence>
}

function Field({ children }: { children: React.ReactNode }) {
  return <div className="flex items-center gap-3 rounded-2xl border border-white/10 bg-white/5 px-4 py-3">{children}</div>
}

function Select({ v, setV, opts }: { v: string; setV: React.Dispatch<React.SetStateAction<string>> | ((v: string) => void); opts: (string | [string, string])[] }) {
  const rows = opts.map(x => Array.isArray(x) ? x : [x, x])
  return <div className="relative"><select value={v} onChange={e => setV(e.target.value)} className="w-full appearance-none rounded-2xl border border-white/10 bg-white/5 px-4 py-3 text-sm outline-none focus:border-white/20">{rows.map(x => <option key={x[0]} value={x[0]} className="bg-zinc-950">{x[1]}</option>)}</select><ChevronDown className="pointer-events-none absolute right-4 top-1/2 h-4 w-4 -translate-y-1/2 text-zinc-100/90" /></div>
}

function Lab({ t }: { t: string }) { return <label className="mb-2 block text-sm font-medium text-zinc-200">{t}</label> }

function Input({ t, v, onChange, c = '' }: { t: string; v: string; onChange: (v: string) => void; c?: string }) {
  return <div className={c}><Lab t={t} /><input value={v} onChange={e => onChange(e.target.value)} className="w-full rounded-2xl border border-white/10 bg-white/5 px-4 py-3 text-sm outline-none focus:border-white/20" /></div>
}

function CheckRow({ on, setOn, t }: { on: boolean; setOn: (v: boolean) => void; t: string }) {
  return <button onClick={() => setOn(!on)} className={cls('flex items-center justify-between rounded-2xl border px-4 py-3 text-sm', on ? 'border-white/20 bg-white/10' : 'border-white/10 bg-white/5')}><span>{t}</span><span className={cls('h-5 w-9 rounded-full p-0.5 transition', on ? 'bg-white' : 'bg-zinc-700')}><span className={cls('block h-4 w-4 rounded-full transition', on ? 'translate-x-4 bg-zinc-900' : 'translate-x-0 bg-white')} /></span></button>
}
