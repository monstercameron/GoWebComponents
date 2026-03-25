export default function DynamicCatalog(parseItems) {
  return (
    <main>
      {parseItems.map(item => <section>{item.title}</section>)}
    </main>
  );
}
