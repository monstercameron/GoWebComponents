$ErrorActionPreference='Stop'
$db = Join-Path $repo 'docs\docs.db'
$schema = Join-Path $repo 'docs\docs.sql'
$paths = @(
  'docs/SERVER_FUNCTIONS.md',
  'docs/INTEROP.md',
  'docs/STATE_ARCHITECTURE.md',
  'docs/PERFORMANCE.md',
  'docs/WALKTHROUGHS.md',
  'docs/PWA.md',
  'docs/benchmarks/reference.json',
  'docs/STARTER_OUTPUT_RULES.md',
  'docs/MIGRATIONS.md',
  'docs/deletemesoon/example-preview.html'
)

if (Test-Path $db) { Remove-Item -Force $db }
& sqlite3 $db ".read `"$schema`""

function Escape-Sql([string]$v){
  if ($null -eq $v) { return '' }
  return $v.Replace("'", "''")
}
function Slugify([string]$v){
  if ([string]::IsNullOrWhiteSpace($v)) { return $null }
  $s = $v.ToLowerInvariant()
  $s = [regex]::Replace($s, '[^a-z0-9]+', '-')
  $s = $s.Trim('-')
  if ($s.Length -gt 80) { $s = $s.Substring(0,80) }
  return $s
}

$docId=1
$orders = @{}
$orders[$docId]=0

$insertSql = @()

foreach($path in $paths){
  $full = Join-Path $repo $path
  $txt = Get-Content -Raw -Path $full
  $ext = [IO.Path]::GetExtension($path).ToLower()
  $kind = switch($ext){'.md'{'md'} '.json'{'json'} '.html'{'html'} default{'other'}}
  $status = if($path -like 'docs/deletemesoon/*'){ 'archived' } else { 'shipped' }

  if($ext -eq '.md' -and $txt -match '(?m)^#\s+(.+)$') { $title = $matches[1] }
  elseif($ext -eq '.html' -and $txt -match '(?s)<title>(.*?)</title>'){ $title = $matches[1] }
  elseif($ext -eq '.json'){ $title = 'Benchmark Reference Snapshot' }
  else { $title = [IO.Path]::GetFileNameWithoutExtension($path) }

  $meta = @{ updated=(Get-Date).ToString('o'); words=(($txt -replace "`r",'').Trim().Split([char[]]@( ' ',"`t","`n","`r"), [StringSplitOptions]::RemoveEmptyEntries).Count); chars=$txt.Length }
  $insertSql += "INSERT INTO docs_document (id,path,title,kind,status,meta_json) VALUES ($docId,'$(Escape-Sql $path)','$(Escape-Sql $title)','$kind','$status','$(Escape-Sql ($meta | ConvertTo-Json -Compress))');"
  $orders[$docId] = 1

  if($ext -eq '.json') { $docId++; continue }

  if($ext -eq '.html'){
    $insertSql += "INSERT INTO docs_node(document_id,node_type,level,title,body_text,payload_json,sort_order,section_anchor) VALUES ($docId,'title',1,'$(Escape-Sql $title)','$(Escape-Sql 'HTML preview shell and React runtime bootstrap page')','$(Escape-Sql ('{`"node`":`"html`"}') )', $(($orders[$docId]++)); ';
    continue
  }

  $sections = Select-String -InputObject $txt -Pattern '^(#{1,6})\s+(.+)$' -AllMatches
  if($sections.Matches.Count -eq 0){
    $insertSql += "INSERT INTO docs_node(document_id,node_type,level,title,body_text,payload_json,sort_order,section_anchor) VALUES ($docId,'paragraph',2,NULL,'$(Escape-Sql ($txt.Trim()))','$(Escape-Sql '{`"node`":`"paragraph`"}')', $($orders[$docId]++));"
    $docId++
    continue
  }

  $blocks = @()
  for($i=0; $i -lt $sections.Matches.Count; $i++){
    $m = $sections.Matches[$i]
    $start = $m.Index + $m.Length
    $end = if($i+1 -lt $sections.Matches.Count){ $sections.Matches[$i+1].Index } else { $txt.Length }
    $chunk = $txt.Substring($start, $end - $start)
    $blocks += [pscustomobject]@{Level=$m.Groups[1].Value.Length; Title=$m.Groups[2].Value.Trim(); Chunk=$chunk}
  }

  $isFirst = $true
  for($i=0; $i -lt $blocks.Count; $i++){
    $b = $blocks[$i]
    $nodeType = if($b.Level -eq 1){ 'heading' } else { 'section' }
    $anchor = Slugify $b.Title
    $payload = @{ level=$b.Level; source='markdown' } | ConvertTo-Json -Compress
    $insertSql += "INSERT INTO docs_node(document_id,node_type,level,title,body_text,payload_json,sort_order,section_anchor) VALUES ($docId,'$nodeType',$($b.Level),'$(Escape-Sql $b.Title)',NULL,'$(Escape-Sql $payload)', $($orders[$docId]++), '$(Escape-Sql $anchor)');"

    $chunk = $b.Chunk

    # Code blocks
    $codeMatches = [regex]::Matches($chunk,'(?s)^```([a-zA-Z0-9_-]*)\r?\n(.*?)^```\s*$','Multiline')
    $ci = 1
    foreach($cm in $codeMatches){
      $lang = if([string]::IsNullOrWhiteSpace($cm.Groups[1].Value)){'text'} else {$cm.Groups[1].Value}
      $code = $cm.Groups[2].Value.TrimEnd()
      if($code.Length -gt 600){ $code = $code.Substring(0,600) }
      $codePayload = @{ language=$lang; lines=($code -split "`n").Count; idx=$ci } | ConvertTo-Json -Compress
      $insertSql += "INSERT INTO docs_node(document_id,node_type,level,title,body_text,payload_json,sort_order,section_anchor) VALUES ($docId,'code',3,'Code $ci','$(Escape-Sql $code)','$(Escape-Sql $codePayload)', $($orders[$docId]++), '$(Escape-Sql $anchor)');"
      $ci++
    }

    # Checklist rows
    foreach($line in ($chunk -split "`n")){
      if($line -match '^\s*[-*]\s+\[[ xX]?\]\s+(?<x>.+)$'){
        $item = $matches['x'].Trim()
        $isChecked = $line -match '\[[xX]\]'
        $payload = @{ checked=[bool]$isChecked } | ConvertTo-Json -Compress
        $insertSql += "INSERT INTO docs_node(document_id,node_type,level,title,body_text,payload_json,sort_order,section_anchor) VALUES ($docId,'checklist',3,NULL,'$(Escape-Sql $item)','$(Escape-Sql $payload)', $($orders[$docId]++), '$(Escape-Sql $anchor)');"
      }
    }

    # Links rows
    $linkMatches = [regex]::Matches($chunk,'\[([^\]]+)\]\(([^)]+)\)')
    foreach($lm in $linkMatches){
      $text = $lm.Groups[1].Value
      $url = $lm.Groups[2].Value
      $payload = @{ text=$text; url=$url } | ConvertTo-Json -Compress
      $insertSql += "INSERT INTO docs_node(document_id,node_type,level,title,body_text,payload_json,sort_order,section_anchor) VALUES ($docId,'link',3,'$(Escape-Sql $text)',NULL,'$(Escape-Sql $payload)', $($orders[$docId]++), '$(Escape-Sql $anchor)');"
    }

    # Paragraph summary
    $plain = $chunk -replace '(?ms)^```.*?^```\s*$',''
    $plain = [regex]::Replace($plain,'\[([^\]]+)\]\([^)]+\)','').Trim()
    $plain = [regex]::Replace($plain,'\s+',' ')
    if($plain.Length -gt 0){
      if($plain.Length -gt 700){ $plain = $plain.Substring(0,700) }
      $pp = @{ words=($plain -split '\s+').Count } | ConvertTo-Json -Compress
      $insertSql += "INSERT INTO docs_node(document_id,node_type,level,title,body_text,payload_json,sort_order,section_anchor) VALUES ($docId,'paragraph',2,NULL,'$(Escape-Sql $plain)','$(Escape-Sql $pp)', $($orders[$docId]++), '$(Escape-Sql $anchor)');"
    }
  }

  $docId++
}

$benchText = Get-Content -Raw -Path (Join-Path $repo 'docs/benchmarks/reference.json')
$insertSql += "INSERT INTO docs_run(document_id, source_kind, run_key, source_path, generated_at_utc, run_meta_json, payload_json) VALUES ((SELECT id FROM docs_document WHERE path='docs/benchmarks/reference.json'),'benchmark_json','bench-ref-2026-03-25','docs/benchmarks/reference.json','2026-03-25T15:17:41Z','$(Escape-Sql '{`"ingested`":true}')','$(Escape-Sql $benchText)');"

# execute
$seedFile = Join-Path $repo 'docs\_seed_rows.sql'
Set-Content -Path $seedFile -Value ($insertSql -join "`n") -Encoding UTF8
& sqlite3 $db < $seedFile
& sqlite3 $db 'SELECT "docs_document" AS t, COUNT(*) FROM docs_document; SELECT "docs_node" AS t, COUNT(*) FROM docs_node; SELECT "docs_run" AS t, COUNT(*) FROM docs_run;'
