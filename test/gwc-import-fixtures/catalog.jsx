export function CatalogPage() {
  return (
    <main
      data-view="catalog"
      data-region="eu"
      aria-live="polite"
      aria-busy={false}
      style={{ backgroundColor: "#101820", color: "#f6f8fb", paddingTop: 24, minHeight: "100vh" }}
    >
      <header data-surface="masthead">
        <nav>
          <a href="/shop">Shop</a>
          <a href="/deals">Deals</a>
        </nav>
      </header>
      <article data-track="featured">
        <h1>Catalog</h1>
        <input type="checkbox" checked={true} />
        <span>{42}</span>
        <price-badge tone="sale" priority={2} featured={true}>Sale</price-badge>
        <inventory-pill>Sold out</inventory-pill>
        <marketing-card>Built for teams</marketing-card>
        <promo-callout>Terms apply.</promo-callout>
      </article>
      <footer>Thanks for visiting.</footer>
    </main>
  );
}
