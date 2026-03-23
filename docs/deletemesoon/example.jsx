import React, { useEffect, useMemo, useState } from 'react';

const MODULES = ['all', 'core', 'router', 'state', 'cli', 'data', 'rendering', 'design', 'plugins', 'forms', 'motion', 'commerce'];
const STATUSES = ['all', 'stable', 'experimental', 'deprecated'];
const LEVELS = ['all', 'Beginner', 'Core', 'Intermediate', 'Advanced'];
const SORT_OPTIONS = [
  { value: 'relevance', label: 'Relevance' },
  { value: 'alpha', label: 'A–Z' },
  { value: 'level', label: 'Level' },
];

const ITEMS = [
  {
    id: 1,
    title: 'Getting Started',
    status: 'stable',
    module: 'core',
    type: 'Concept',
    level: 'Beginner',
    tags: ['setup', 'intro'],
    blurb: 'Project overview, install flow, and first render path.',
    readTime: '4 min read',
    content: {
      kind: 'article',
      sections: [
        {
          heading: 'What this is',
          paragraphs: [
            'Concept pages are lightweight markdown-style write-ups that explain the why behind the framework. They should read like concise articles rather than reference tables.',
            'For a docs site on GitHub Pages, this content can be authored as simple MD files and rendered into a polished article shell with headings, callouts, and code snippets.',
          ],
        },
        {
          heading: 'Recommended flow',
          paragraphs: [
            'Start with installation, show the minimum project structure, then explain the first successful render. Keep the first concept page narrative and fast to scan.',
            'Use these pages to teach mental models, not to enumerate every option. That job belongs to the structured API docs.',
          ],
        },
      ],
      callout: 'A concept page should feel like a short engineering blog post with tight prose and a clear takeaway.',
      code: `# Install\nnpm install your-project\n\n# Run locally\nnpm run dev\n\n# Deploy to GitHub Pages\nnpm run build`,
    },
  },
  {
    id: 2,
    title: 'Component API',
    status: 'stable',
    module: 'core',
    type: 'API',
    level: 'Core',
    tags: ['components', 'props'],
    blurb: 'Reference for core UI primitives and composition patterns.',
    readTime: 'Reference',
    content: {
      kind: 'api',
      signature: 'createComponent(name: string, render: RenderFn, options?: ComponentOptions): ComponentDefinition',
      summary: 'Defines a reusable UI unit with typed props, render behavior, and optional metadata for docs generation.',
      params: [
        { name: 'name', type: 'string', required: 'yes', description: 'Stable component identifier used in tooling and dev warnings.' },
        { name: 'render', type: 'RenderFn', required: 'yes', description: 'Pure render function that returns the component view.' },
        { name: 'options', type: 'ComponentOptions', required: 'no', description: 'Extra metadata such as display name, slots, and docs hints.' },
      ],
      returns: 'ComponentDefinition',
      example: `const Button = createComponent('Button', (props) => {\n  return <button>{props.label}</button>;\n});`,
      notes: [
        'API pages should be highly structured and easy to scan.',
        'Use signatures, param tables, return types, examples, and edge-case notes.',
      ],
    },
  },
  {
    id: 3,
    title: 'State Hooks',
    status: 'stable',
    module: 'state',
    type: 'API',
    level: 'Core',
    tags: ['state', 'hooks'],
    blurb: 'State helpers, lifecycle semantics, and mutation rules.',
    readTime: 'Reference',
    content: {
      kind: 'api',
      signature: 'useState<T>(initial: T | (() => T)): [T, (next: T | ((prev: T) => T)) => void]',
      summary: 'Creates local reactive state for a component and returns the current value plus an updater function.',
      params: [
        { name: 'initial', type: 'T | (() => T)', required: 'yes', description: 'Initial state value or lazy initializer.' },
      ],
      returns: '[T, setState]',
      example: `const [count, setCount] = useState(0);\nsetCount((prev) => prev + 1);`,
      notes: [
        'Prefer updater functions when the next value depends on previous state.',
        'Keep state local until multiple routes or modules must coordinate.',
      ],
    },
  },
  {
    id: 4,
    title: 'Counter Demo',
    status: 'stable',
    module: 'state',
    type: 'Example',
    level: 'Beginner',
    tags: ['interactive', 'counter'],
    blurb: 'A tiny reactive demo showing local state, derived labels, and click feedback.',
    readTime: 'Interactive',
    content: {
      kind: 'counter',
      description: 'Example pages should be interactive. This simple counter proves the right pane can host live demos rather than static screenshots.',
      tips: ['Uses local state', 'Shows click interactions', 'Fits nicely into a docs demo frame'],
    },
  },
  {
    id: 5,
    title: 'SSR Walkthrough',
    status: 'experimental',
    module: 'rendering',
    type: 'Concept',
    level: 'Advanced',
    tags: ['ssr', 'rendering'],
    blurb: 'How server rendering, hydration, and data boundaries fit together.',
    readTime: '7 min read',
    content: {
      kind: 'article',
      sections: [
        {
          heading: 'Rendering model',
          paragraphs: [
            'SSR concept pages should explain the sequence clearly: render on the server, ship HTML, reattach behavior, then transition into client-side updates.',
            'Keep diagrams optional, but the prose should still stand on its own for engineers who are skimming in a browser tab.',
          ],
        },
        {
          heading: 'Why this belongs in Concepts',
          paragraphs: [
            'This is not a reference entry. It is a mental model page. Readers need sequencing, tradeoffs, and pitfalls more than a parameter table.',
          ],
        },
      ],
      callout: 'Concept docs should answer how the system thinks before API docs answer how the function is called.',
      code: `renderToString(app)\n  -> send HTML\n  -> load client bundle\n  -> hydrate islands\n  -> handle client updates`,
    },
  },
  {
    id: 6,
    title: 'Forms Example',
    status: 'stable',
    module: 'forms',
    type: 'Example',
    level: 'Core',
    tags: ['forms', 'validation'],
    blurb: 'Controlled inputs, submission flow, and optimistic UI patterns.',
    readTime: 'Interactive',
    content: {
      kind: 'counter',
      description: 'This slot can later become a real form demo. For now it reuses the interactive example surface so the docs shell supports live widgets consistently.',
      tips: ['Swap in a real form later', 'Keep example panes interactive', 'Good place for validation demos'],
    },
  },
  {
    id: 7,
    title: 'Data Fetching',
    status: 'experimental',
    module: 'data',
    type: 'API',
    level: 'Advanced',
    tags: ['data', 'async'],
    blurb: 'Query helpers, loading states, cache rules, and invalidation.',
    readTime: 'Reference',
    content: {
      kind: 'api',
      signature: 'useQuery<T>(key: string, loader: () => Promise<T>, options?: QueryOptions<T>): QueryState<T>',
      summary: 'Fetches remote data, tracks loading and error state, and exposes cache-aware refetch behavior.',
      params: [
        { name: 'key', type: 'string', required: 'yes', description: 'Stable cache key for the request.' },
        { name: 'loader', type: '() => Promise<T>', required: 'yes', description: 'Async function that resolves the requested data.' },
        { name: 'options', type: 'QueryOptions<T>', required: 'no', description: 'Controls stale time, retries, background refresh, and selection.' },
      ],
      returns: 'QueryState<T>',
      example: `const users = useQuery('users', () => fetch('/api/users').then((r) => r.json()));`,
      notes: [
        'Expose stale/loading/error states clearly in the docs.',
        'Show cache invalidation rules beside the main signature.',
      ],
    },
  },
  {
    id: 8,
    title: 'Animation Recipes',
    status: 'stable',
    module: 'motion',
    type: 'Example',
    level: 'Intermediate',
    tags: ['motion', 'ui'],
    blurb: 'Hover cards, reveal transitions, and subtle motion systems.',
    readTime: 'Interactive',
    content: {
      kind: 'counter',
      description: 'Animation examples can also live inside this interactive surface. The shell is ready for richer demos later, but already demonstrates reactive behavior.',
      tips: ['Use this for micro-interactions', 'Embed live snippets', 'Pair with code and controls'],
    },
  },
  {
    id: 9,
    title: 'Theming Model',
    status: 'stable',
    module: 'design',
    type: 'Concept',
    level: 'Core',
    tags: ['theme', 'design'],
    blurb: 'Color tokens, dark mode strategy, and visual customization.',
    readTime: '5 min read',
    content: {
      kind: 'article',
      sections: [
        {
          heading: 'Token-first styling',
          paragraphs: [
            'Theming docs should explain the token layers: semantic roles, component aliases, and raw palette values.',
            'Readers should understand where dark mode switches happen and how to extend themes without breaking contrast rules.',
          ],
        },
        {
          heading: 'Docs implication',
          paragraphs: [
            'Because concepts are markdown-like, they are perfect for showing rationale, constraints, and migration guidance rather than raw API surface area.',
          ],
        },
      ],
      callout: 'Use concept pages to document systems thinking and API pages to document call signatures.',
      code: `theme = {\n  surface: 'slate-950',\n  text: 'slate-100',\n  accent: 'cyan-400'\n}`,
    },
  },
  {
    id: 10,
    title: 'CLI Commands',
    status: 'stable',
    module: 'cli',
    type: 'API',
    level: 'Beginner',
    tags: ['cli', 'tooling'],
    blurb: 'Create, build, dev, export, and deployment command surface.',
    readTime: 'Reference',
    content: {
      kind: 'api',
      signature: 'project-cli <command> [...flags]',
      summary: 'Entrypoint for local development, building, previewing, and static deployment workflows.',
      params: [
        { name: 'command', type: 'string', required: 'yes', description: 'One of dev, build, preview, export, or deploy.' },
        { name: 'flags', type: 'string[]', required: 'no', description: 'Optional command-specific flags and overrides.' },
      ],
      returns: 'CLI process exit code',
      example: `project-cli build --out-dir dist\nproject-cli deploy --provider github-pages`,
      notes: [
        'Keep CLI docs compact and task-oriented.',
        'Use command blocks and flag tables rather than long prose.',
      ],
    },
  },
  {
    id: 11,
    title: 'Commerce Demo',
    status: 'deprecated',
    module: 'commerce',
    type: 'Example',
    level: 'Advanced',
    tags: ['dashboard', 'commerce'],
    blurb: 'A richer end-to-end sample with inventory, routes, and widgets.',
    readTime: 'Interactive',
    content: {
      kind: 'counter',
      description: 'Eventually this can be a real commerce demo. For now the example card proves the right pane can host an active widget and supporting notes.',
      tips: ['Can expand into full route demo', 'Supports controls and state', 'Good for end-to-end examples'],
    },
  },
  {
    id: 12,
    title: 'Plugin System',
    status: 'experimental',
    module: 'plugins',
    type: 'Concept',
    level: 'Advanced',
    tags: ['plugins', 'extensibility'],
    blurb: 'Extension lifecycle, capabilities, and registration patterns.',
    readTime: '6 min read',
    content: {
      kind: 'article',
      sections: [
        {
          heading: 'Extension boundaries',
          paragraphs: [
            'Plugin concept pages should define what third parties are allowed to hook into, what remains private, and how compatibility is preserved over time.',
            'These pages work best as prose-first documentation with a few small examples rather than giant schemas.',
          ],
        },
        {
          heading: 'Authoring guidance',
          paragraphs: [
            'Explain lifecycle order, capabilities, and versioning strategy in human language first. Then link out to APIs for the exact interfaces.',
          ],
        },
      ],
      callout: 'When docs readers are making architecture decisions, a clean concept article is more valuable than a raw dump of types.',
      code: `registerPlugin({\n  name: 'analytics',\n  setup(ctx) {\n    ctx.hooks.onRouteChange(() => {});\n  }\n})`,
    },
  },
];

const FILTERS = ['All', 'Concept', 'API', 'Example'];

const BADGE_CLASS = {
  Concept: 'bg-cyan-500/15 text-cyan-200 border-cyan-400/25',
  API: 'bg-violet-500/15 text-violet-200 border-violet-400/25',
  Example: 'bg-emerald-500/15 text-emerald-200 border-emerald-400/25',
};

function filterItems(items, query, activeFilter, statusFilter, levelFilter, moduleFilter) {
  const normalizedQuery = query.trim().toLowerCase();

  return items.filter((item) => {
    const typeMatches = activeFilter === 'All' || item.type === activeFilter;
    const statusMatches = statusFilter === 'all' || item.status === statusFilter;
    const levelMatches = levelFilter === 'all' || item.level === levelFilter;
    const moduleMatches = moduleFilter === 'all' || item.module === moduleFilter;
    const searchableText = [
      item.title,
      item.type,
      item.level,
      item.status,
      item.module,
      item.blurb,
      item.readTime,
      ...item.tags,
    ]
      .join(' ')
      .toLowerCase();
    const queryMatches = normalizedQuery.length === 0 || searchableText.includes(normalizedQuery);
    return typeMatches && statusMatches && levelMatches && moduleMatches && queryMatches;
  });
}

function sortItems(items, sortBy) {
  const levelRank = { Beginner: 0, Core: 1, Intermediate: 2, Advanced: 3 };
  const sorted = [...items];

  if (sortBy === 'alpha') {
    sorted.sort((a, b) => a.title.localeCompare(b.title));
    return sorted;
  }

  if (sortBy === 'level') {
    sorted.sort((a, b) => {
      const levelDiff = (levelRank[a.level] ?? 999) - (levelRank[b.level] ?? 999);
      if (levelDiff !== 0) return levelDiff;
      return a.title.localeCompare(b.title);
    });
    return sorted;
  }

  return sorted;
}

function getContentKindLabel(item) {
  if (!item?.content?.kind) return 'Unknown';
  if (item.content.kind === 'article') return 'Markdown article';
  if (item.content.kind === 'api') return 'Structured API';
  if (item.content.kind === 'counter') return 'Interactive demo';
  return 'Unknown';
}

const FILTER_TEST_CASES = [
  {
    name: 'returns all items when query is empty and filter is All',
    expectedCount: ITEMS.length,
    actualCount: filterItems(ITEMS, '', 'All', 'all', 'all', 'all').length,
  },
  {
    name: 'finds API items by type filter',
    expectedCount: ITEMS.filter((item) => item.type === 'API').length,
    actualCount: filterItems(ITEMS, '', 'API', 'all', 'all', 'all').length,
  },
  {
    name: 'matches tags and blurbs case-insensitively',
    expectedCount: 1,
    actualCount: filterItems(ITEMS, '', 'All', 'deprecated', 'all', 'commerce').length,
  },
  {
    name: 'returns zero for an unmatched query',
    expectedCount: 0,
    actualCount: filterItems(ITEMS, 'no-such-item', 'All', 'all', 'all', 'all').length,
  },
  {
    name: 'filters concepts by text and type together',
    expectedCount: 1,
    actualCount: filterItems(ITEMS, 'theme', 'Concept', 'all', 'Core', 'design').length,
  },
  {
    name: 'trimmed query still matches',
    expectedCount: 0,
    actualCount: filterItems(ITEMS, '  router  ', 'Example', 'all', 'all', 'all').length,
  },
  {
    name: 'module filter narrows results',
    expectedCount: 1,
    actualCount: filterItems(ITEMS, '', 'API', 'all', 'all', 'cli').length,
  },
  {
    name: 'status filter narrows results',
    expectedCount: 3,
    actualCount: filterItems(ITEMS, '', 'All', 'experimental', 'all', 'all').length,
  },
  {
    name: 'level sort starts with beginner',
    expectedCount: 'Getting Started',
    actualCount: sortItems(filterItems(ITEMS, '', 'All', 'all', 'all', 'all'), 'level')[0]?.title,
  },
  {
    name: 'concept entries map to article content',
    expectedCount: 'Markdown article',
    actualCount: getContentKindLabel(ITEMS.find((item) => item.id === 1)),
  },
  {
    name: 'api entries map to structured docs',
    expectedCount: 'Structured API',
    actualCount: getContentKindLabel(ITEMS.find((item) => item.id === 2)),
  },
  {
    name: 'example entries map to interactive demos',
    expectedCount: 'Interactive demo',
    actualCount: getContentKindLabel(ITEMS.find((item) => item.id === 4)),
  },
];

function StatCard({ value, label }) {
  return (
    <div className="rounded-3xl border border-white/10 bg-white/5 p-3 shadow-lg shadow-black/10">
      <div className="text-xl font-semibold text-white">{value}</div>
      <div className="mt-1 text-[11px] uppercase tracking-[0.18em] text-slate-400">{label}</div>
    </div>
  );
}

function ItemCard({ item, isActive, onSelect }) {
  const statusClass = item.status === 'stable'
    ? 'border-emerald-400/20 bg-emerald-400/10 text-emerald-200'
    : item.status === 'experimental'
      ? 'border-amber-400/20 bg-amber-400/10 text-amber-200'
      : 'border-rose-400/20 bg-rose-400/10 text-rose-200';

  return (
    <button
      type="button"
      onClick={() => onSelect(item.id)}
      className={[
        'group w-full cursor-pointer rounded-[22px] border p-3 text-left transition duration-200',
        isActive
          ? 'border-cyan-300/35 bg-cyan-400/10 shadow-xl shadow-cyan-950/25'
          : 'border-white/10 bg-white/[0.04] hover:-translate-y-0.5 hover:border-white/20 hover:bg-white/[0.08]',
      ].join(' ')}
    >
      <div className="flex items-start justify-between gap-2">
        <div className="min-w-0 flex-1">
          <div className="text-sm font-medium text-white">{item.title}</div>
          <div className="mt-1 line-clamp-2 text-xs leading-5 text-slate-400">{item.blurb}</div>
        </div>
        <span
          className={[
            'shrink-0 rounded-full border px-2 py-1 text-[10px] font-medium uppercase tracking-[0.14em]',
            BADGE_CLASS[item.type],
          ].join(' ')}
        >
          {item.type}
        </span>
      </div>
      <div className="mt-3 flex flex-wrap items-center gap-1">
        <span className={['rounded-full border px-2 py-1 text-[10px] uppercase tracking-[0.14em]', statusClass].join(' ')}>
          {item.status}
        </span>
        <span className="rounded-full border border-white/10 bg-black/20 px-2 py-1 text-[10px] uppercase tracking-[0.14em] text-slate-400">
          {item.level}
        </span>
        <span className="rounded-full border border-white/10 bg-white/5 px-2 py-1 text-[10px] text-slate-300">
          {item.module}
        </span>
        <span className="rounded-full border border-white/10 bg-white/5 px-2 py-1 text-[10px] text-slate-300">
          {item.readTime}
        </span>
        {item.tags.map((tag) => (
          <span key={tag} className="rounded-full border border-white/10 bg-white/5 px-2 py-1 text-[10px] text-slate-300">
            #{tag}
          </span>
        ))}
      </div>
    </button>
  );
}

function ConceptArticle({ item }) {
  return (
    <div className="flex min-h-full flex-col rounded-[22px] border border-white/10 bg-slate-950/35 p-4 shadow-inner shadow-black/20">
      <div className="border-b border-white/10 pb-3">
        <div className="text-sm font-medium text-white">Concept article</div>
        <div className="text-xs uppercase tracking-[0.18em] text-slate-500">Markdown-style write-up</div>
      </div>

      <div className="mt-4 flex flex-1 flex-col gap-4">
        <div className="rounded-[22px] border border-cyan-400/20 bg-cyan-400/10 p-4 text-sm leading-7 text-cyan-50">
          {item.content.callout}
        </div>

        {item.content.sections.map((section) => (
          <article key={section.heading} className="rounded-[20px] border border-white/10 bg-white/[0.04] p-4">
            <h3 className="text-lg font-semibold text-white">{section.heading}</h3>
            <div className="mt-3 space-y-3 text-sm leading-7 text-slate-300">
              {section.paragraphs.map((paragraph) => (
                <p key={paragraph}>{paragraph}</p>
              ))}
            </div>
          </article>
        ))}

        <div className="rounded-[20px] border border-white/10 bg-[#06101d] p-4">
          <div className="text-xs uppercase tracking-[0.18em] text-slate-500">Example markdown block</div>
          <pre className="mt-3 overflow-x-auto text-sm leading-6 text-cyan-100">
            <code>{item.content.code}</code>
          </pre>
        </div>
      </div>
    </div>
  );
}

function ApiReference({ item }) {
  return (
    <div className="flex min-h-full flex-col rounded-[22px] border border-white/10 bg-slate-950/35 p-4 shadow-inner shadow-black/20">
      <div className="border-b border-white/10 pb-3">
        <div className="text-sm font-medium text-white">API reference</div>
        <div className="text-xs uppercase tracking-[0.18em] text-slate-500">Structured documentation</div>
      </div>

      <div className="mt-4 flex flex-1 flex-col gap-4">
        <div className="rounded-[22px] border border-violet-400/20 bg-violet-400/10 p-4">
          <div className="text-xs uppercase tracking-[0.18em] text-violet-200">Signature</div>
          <pre className="mt-3 overflow-x-auto text-sm leading-6 text-violet-50">
            <code>{item.content.signature}</code>
          </pre>
          <p className="mt-3 text-sm leading-7 text-slate-200">{item.content.summary}</p>
        </div>

        <div className="rounded-[20px] border border-white/10 bg-white/[0.04] p-4">
          <div className="text-sm font-medium text-white">Parameters</div>
          <div className="mt-4 overflow-x-auto">
            <table className="min-w-full text-left text-sm">
              <thead>
                <tr className="border-b border-white/10 text-slate-400">
                  <th className="pb-2 pr-4 font-medium">Name</th>
                  <th className="pb-2 pr-4 font-medium">Type</th>
                  <th className="pb-2 pr-4 font-medium">Required</th>
                  <th className="pb-2 font-medium">Description</th>
                </tr>
              </thead>
              <tbody>
                {item.content.params.map((param) => (
                  <tr key={param.name} className="border-b border-white/5 align-top text-slate-300 last:border-b-0">
                    <td className="py-3 pr-4 font-medium text-white">{param.name}</td>
                    <td className="py-3 pr-4 text-cyan-200">{param.type}</td>
                    <td className="py-3 pr-4 uppercase">{param.required}</td>
                    <td className="py-3">{param.description}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </div>

        <div className="grid gap-3 md:grid-cols-2">
          <div className="rounded-[20px] border border-white/10 bg-white/[0.04] p-4">
            <div className="text-sm font-medium text-white">Returns</div>
            <div className="mt-3 rounded-xl border border-white/10 bg-black/15 px-3 py-2 text-sm text-emerald-200">
              {item.content.returns}
            </div>
          </div>
          <div className="rounded-[20px] border border-white/10 bg-white/[0.04] p-4">
            <div className="text-sm font-medium text-white">Notes</div>
            <ul className="mt-3 space-y-2 text-sm leading-6 text-slate-300">
              {item.content.notes.map((note) => (
                <li key={note}>{note}</li>
              ))}
            </ul>
          </div>
        </div>

        <div className="rounded-[20px] border border-white/10 bg-[#06101d] p-4">
          <div className="text-xs uppercase tracking-[0.18em] text-slate-500">Usage example</div>
          <pre className="mt-3 overflow-x-auto text-sm leading-6 text-cyan-100">
            <code>{item.content.example}</code>
          </pre>
        </div>
      </div>
    </div>
  );
}

function CounterExample({ item }) {
  const [count, setCount] = useState(0);

  const tone = count === 0 ? 'Ready' : count > 0 ? 'Positive' : 'Negative';

  return (
    <div className="flex min-h-full flex-col rounded-[22px] border border-white/10 bg-slate-950/35 p-4 shadow-inner shadow-black/20">
      <div className="border-b border-white/10 pb-3">
        <div className="text-sm font-medium text-white">Interactive example</div>
        <div className="text-xs uppercase tracking-[0.18em] text-slate-500">Reactive demo surface</div>
      </div>

      <div className="mt-4 grid flex-1 gap-3 lg:grid-cols-[minmax(0,1.4fr)_280px]">
        <div className="rounded-[22px] border border-emerald-400/20 bg-emerald-400/10 p-5">
          <div className="text-xs uppercase tracking-[0.18em] text-emerald-200">Live widget</div>
          <div className="mt-4 text-5xl font-semibold tracking-tight text-white">{count}</div>
          <div className="mt-2 text-sm text-emerald-50/90">State tone: {tone}</div>

          <div className="mt-5 flex flex-wrap gap-2">
            <button
              type="button"
              onClick={() => setCount((prev) => prev - 1)}
              className="rounded-2xl border border-white/15 bg-black/20 px-4 py-2 text-sm font-medium text-white transition hover:bg-black/30"
            >
              Decrement
            </button>
            <button
              type="button"
              onClick={() => setCount((prev) => prev + 1)}
              className="rounded-2xl border border-white/15 bg-black/20 px-4 py-2 text-sm font-medium text-white transition hover:bg-black/30"
            >
              Increment
            </button>
            <button
              type="button"
              onClick={() => setCount(0)}
              className="rounded-2xl border border-white/15 bg-black/20 px-4 py-2 text-sm font-medium text-white transition hover:bg-black/30"
            >
              Reset
            </button>
          </div>

          <p className="mt-5 text-sm leading-7 text-emerald-50/90">{item.content.description}</p>
        </div>

        <div className="space-y-3">
          <div className="rounded-[20px] border border-white/10 bg-white/[0.04] p-4">
            <div className="text-sm font-medium text-white">Why this matters</div>
            <ul className="mt-3 space-y-2 text-sm leading-6 text-slate-300">
              {item.content.tips.map((tip) => (
                <li key={tip}>{tip}</li>
              ))}
            </ul>
          </div>
          <div className="rounded-[20px] border border-white/10 bg-[#06101d] p-4">
            <div className="text-xs uppercase tracking-[0.18em] text-slate-500">Example source</div>
            <pre className="mt-3 overflow-x-auto text-sm leading-6 text-cyan-100">
              <code>{`const [count, setCount] = useState(0);\n\n<button onClick={() => setCount((prev) => prev + 1)}>\n  Increment\n</button>`}</code>
            </pre>
          </div>
        </div>
      </div>
    </div>
  );
}

function DisplaySurface({ item }) {
  if (!item) {
    return (
      <div className="flex min-h-full items-center justify-center rounded-[22px] border border-dashed border-white/10 bg-black/10 p-8 text-sm text-slate-400">
        Nothing selected.
      </div>
    );
  }

  if (item.content.kind === 'article') {
    return <ConceptArticle item={item} />;
  }

  if (item.content.kind === 'api') {
    return <ApiReference item={item} />;
  }

  return <CounterExample item={item} />;
}

export default function DocsDemosSiteConcept() {
  const [query, setQuery] = useState('');
  const [activeFilter, setActiveFilter] = useState('All');
  const [statusFilter, setStatusFilter] = useState('all');
  const [levelFilter, setLevelFilter] = useState('all');
  const [moduleFilter, setModuleFilter] = useState('all');
  const [sortBy, setSortBy] = useState('relevance');
  const [selectedId, setSelectedId] = useState(ITEMS[0]?.id ?? null);

  const filteredItems = useMemo(() => {
    const filtered = filterItems(ITEMS, query, activeFilter, statusFilter, levelFilter, moduleFilter);
    return sortItems(filtered, sortBy);
  }, [query, activeFilter, statusFilter, levelFilter, moduleFilter, sortBy]);

  useEffect(() => {
    if (filteredItems.length === 0) {
      return;
    }

    const selectedStillVisible = filteredItems.some((item) => item.id === selectedId);
    if (!selectedStillVisible) {
      setSelectedId(filteredItems[0].id);
    }
  }, [filteredItems, selectedId]);

  const selected = filteredItems.find((item) => item.id === selectedId) ?? filteredItems[0] ?? ITEMS[0] ?? null;

  return (
    <div className="min-h-screen bg-[radial-gradient(circle_at_top_left,rgba(59,130,246,0.20),transparent_28%),radial-gradient(circle_at_top_right,rgba(168,85,247,0.18),transparent_24%),linear-gradient(180deg,#07111f_0%,#091427_40%,#0b1020_100%)] text-slate-100">
      <div className="mx-auto flex min-h-screen max-w-7xl flex-col px-3 py-3 sm:px-4 sm:py-4 lg:px-5">
        <header className="relative overflow-hidden rounded-[24px] border border-white/10 bg-white/5 p-5 shadow-2xl shadow-black/30 backdrop-blur-xl sm:p-6">
          <div className="absolute inset-0 bg-[linear-gradient(135deg,rgba(255,255,255,0.08),transparent_35%,rgba(255,255,255,0.03))]" />
          <div className="relative flex flex-col gap-6 lg:flex-row lg:items-end lg:justify-between">
            <div className="max-w-3xl space-y-3">
              <div className="inline-flex items-center gap-2 rounded-full border border-cyan-400/30 bg-cyan-400/10 px-3 py-1 text-xs font-medium uppercase tracking-[0.24em] text-cyan-200">
                <span className="h-2 w-2 rounded-full bg-cyan-300" />
                Docs • APIs • Demos
              </div>
              <div className="space-y-2">
                <h1 className="text-3xl font-semibold tracking-tight text-white sm:text-4xl lg:text-5xl">
                  ProjectName developer docs with polished live examples
                </h1>
                <p className="max-w-2xl text-sm leading-6 text-slate-300 sm:text-base sm:leading-7">
                  Concepts render as markdown-style write-ups, API entries render as structured reference docs, and examples render as live interactive demos.
                </p>
              </div>
              <div className="flex flex-wrap gap-2">
                <button
                  type="button"
                  onClick={() => setActiveFilter('Example')}
                  className="cursor-pointer rounded-2xl border border-cyan-300/30 bg-cyan-400/15 px-4 py-2 text-sm font-medium text-cyan-100 transition hover:-translate-y-0.5 hover:bg-cyan-400/20 active:translate-y-0"
                >
                  Explore examples
                </button>
                <button
                  type="button"
                  onClick={() => setActiveFilter('API')}
                  className="cursor-pointer rounded-2xl border border-white/10 bg-white/5 px-4 py-2 text-sm font-medium text-slate-100 transition hover:-translate-y-0.5 hover:bg-white/10 active:translate-y-0"
                >
                  Read the API
                </button>
              </div>
            </div>

            <div className="grid grid-cols-2 gap-2 sm:gap-3 lg:w-[320px]">
              <StatCard value="120+" label="Docs nodes" />
              <StatCard value="35" label="Examples" />
              <StatCard value="Fast" label="Search and filter" />
              <StatCard value="100%" label="GitHub Pages" />
            </div>
          </div>
        </header>

        <main className="mt-3 flex flex-1 flex-col gap-3 lg:min-h-0 lg:flex-row">
          <section className="flex min-h-[420px] flex-col rounded-[24px] border border-white/10 bg-white/5 backdrop-blur-xl lg:sticky lg:top-3 lg:h-[calc(100vh-1.5rem)] lg:w-[34%] xl:w-[31%]">
            <div className="sticky top-0 z-10 border-b border-white/10 bg-slate-950/60 p-3 backdrop-blur-xl sm:p-4">
              <div className="flex flex-col gap-2">
                <div className="flex flex-col gap-2 sm:flex-row sm:items-center">
                  <div className="relative flex-1">
                    <input
                      value={query}
                      onChange={(event) => setQuery(event.target.value)}
                      placeholder="Search concepts, APIs, examples..."
                      className="w-full rounded-xl border border-white/10 bg-slate-950/40 px-3 py-2 text-sm text-white outline-none placeholder:text-slate-500 transition focus:border-cyan-300/40 focus:bg-slate-950/60"
                    />
                  </div>
                  <div className="text-xs uppercase tracking-[0.18em] text-slate-400">
                    {filteredItems.length} results
                  </div>
                </div>
                <div className="flex flex-wrap gap-2">
                  {FILTERS.map((filter) => {
                    const isActive = filter === activeFilter;
                    return (
                      <button
                        key={filter}
                        type="button"
                        onClick={() => setActiveFilter(filter)}
                        className={[
                          'cursor-pointer rounded-xl border px-2.5 py-1.5 text-xs transition',
                          isActive
                            ? 'border-cyan-300/40 bg-cyan-400/15 text-cyan-100 shadow-lg shadow-cyan-900/20'
                            : 'border-white/10 bg-white/5 text-slate-300 hover:bg-white/10 hover:text-white',
                        ].join(' ')}
                      >
                        {filter}
                      </button>
                    );
                  })}
                </div>
                <div className="grid grid-cols-2 gap-2 xl:grid-cols-4">
                  <label className="flex flex-col gap-1 text-[11px] uppercase tracking-[0.16em] text-slate-500">
                    <span>Status</span>
                    <select
                      value={statusFilter}
                      onChange={(event) => setStatusFilter(event.target.value)}
                      className="rounded-xl border border-white/10 bg-slate-950/50 px-3 py-2 text-xs text-slate-100 outline-none"
                    >
                      {STATUSES.map((status) => (
                        <option key={status} value={status}>
                          {status}
                        </option>
                      ))}
                    </select>
                  </label>
                  <label className="flex flex-col gap-1 text-[11px] uppercase tracking-[0.16em] text-slate-500">
                    <span>Difficulty</span>
                    <select
                      value={levelFilter}
                      onChange={(event) => setLevelFilter(event.target.value)}
                      className="rounded-xl border border-white/10 bg-slate-950/50 px-3 py-2 text-xs text-slate-100 outline-none"
                    >
                      {LEVELS.map((level) => (
                        <option key={level} value={level}>
                          {level}
                        </option>
                      ))}
                    </select>
                  </label>
                  <label className="flex flex-col gap-1 text-[11px] uppercase tracking-[0.16em] text-slate-500">
                    <span>Module</span>
                    <select
                      value={moduleFilter}
                      onChange={(event) => setModuleFilter(event.target.value)}
                      className="rounded-xl border border-white/10 bg-slate-950/50 px-3 py-2 text-xs text-slate-100 outline-none"
                    >
                      {MODULES.map((moduleName) => (
                        <option key={moduleName} value={moduleName}>
                          {moduleName}
                        </option>
                      ))}
                    </select>
                  </label>
                  <label className="flex flex-col gap-1 text-[11px] uppercase tracking-[0.16em] text-slate-500">
                    <span>Sort</span>
                    <select
                      value={sortBy}
                      onChange={(event) => setSortBy(event.target.value)}
                      className="rounded-xl border border-white/10 bg-slate-950/50 px-3 py-2 text-xs text-slate-100 outline-none"
                    >
                      {SORT_OPTIONS.map((option) => (
                        <option key={option.value} value={option.value}>
                          {option.label}
                        </option>
                      ))}
                    </select>
                  </label>
                </div>
              </div>
            </div>

            <div className="min-h-0 flex-1 overflow-y-auto p-2 sm:p-3">
              <div className="space-y-2">
                {filteredItems.map((item) => (
                  <ItemCard
                    key={item.id}
                    item={item}
                    isActive={selected?.id === item.id}
                    onSelect={setSelectedId}
                  />
                ))}

                {filteredItems.length === 0 && (
                  <div className="rounded-[22px] border border-dashed border-white/10 bg-black/10 p-6 text-center text-sm text-slate-400">
                    No matches yet. Try a broader search or switch the active filter.
                  </div>
                )}
              </div>
            </div>
          </section>

          <section className="flex min-h-[420px] flex-1 flex-col rounded-[24px] border border-white/10 bg-white/5 backdrop-blur-xl lg:sticky lg:top-3 lg:h-[calc(100vh-1.5rem)]">
            <div className="sticky top-0 z-10 border-b border-white/10 bg-slate-950/60 p-4 backdrop-blur-xl sm:p-5">
              {selected ? (
                <>
                  <div className="flex flex-wrap items-center gap-3">
                    <span
                      className={[
                        'rounded-full border px-3 py-1 text-xs font-medium uppercase tracking-[0.2em]',
                        BADGE_CLASS[selected.type],
                      ].join(' ')}
                    >
                      {selected.type}
                    </span>
                    <span className="rounded-full border border-white/10 bg-white/5 px-3 py-1 text-xs uppercase tracking-[0.2em] text-slate-400">
                      {selected.level}
                    </span>
                    <span className="rounded-full border border-white/10 bg-white/5 px-3 py-1 text-xs uppercase tracking-[0.2em] text-slate-400">
                      {getContentKindLabel(selected)}
                    </span>
                  </div>
                  <h2 className="mt-4 text-2xl font-semibold tracking-tight text-white sm:text-3xl">
                    {selected.title}
                  </h2>
                  <p className="mt-3 max-w-3xl text-sm leading-7 text-slate-300 sm:text-base">
                    {selected.blurb}
                  </p>
                </>
              ) : (
                <>
                  <h2 className="text-2xl font-semibold tracking-tight text-white sm:text-3xl">Nothing selected</h2>
                  <p className="mt-3 max-w-3xl text-sm leading-7 text-slate-300 sm:text-base">
                    Adjust the filters or search query to bring results back into view.
                  </p>
                </>
              )}
            </div>

            <div className="min-h-0 flex-1 overflow-y-auto p-3 sm:p-4">
              <DisplaySurface item={selected} />
            </div>
          </section>
        </main>
      </div>
    </div>
  );
}
