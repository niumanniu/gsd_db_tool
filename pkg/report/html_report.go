package report

import (
	"db-diff/pkg/comparator"
	"fmt"
	"html/template"
	"os"
)

// htmlReportData HTML 报告数据结构
type htmlReportData struct {
	SourceSchema   string
	TargetSchema   string
	TableDiff      comparator.TableDiff
	ColumnDiff     map[string]comparator.ColumnDiff
	IndexDiff      map[string]comparator.IndexDiff
	ConstraintDiff map[string]comparator.ConstraintDiff
	TableCounts    map[string]comparator.TableCount
	TableDataDiff  map[string]comparator.DataDiff
	HasDiff        bool
	HasDataDiff    bool
}

// GenerateHTML 生成 HTML 格式报告
func GenerateHTML(result *comparator.DiffResult, outputFile string, verbose bool) error {
	tmplContent := `
<!DOCTYPE html>
<html lang="zh-CN">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>数据库结构对比报告</title>
    <style>
        * { margin: 0; padding: 0; box-sizing: border-box; }
        body { font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif;
               background: #f5f5f5; padding: 20px; line-height: 1.6; }
        .container { max-width: 1200px; margin: 0 auto; }
        h1 { color: #333; padding: 20px; text-align: center; background: #fff;
             border-radius: 8px; margin-bottom: 20px; box-shadow: 0 2px 4px rgba(0,0,0,0.1); }
        .info-panel { background: #fff; padding: 15px 20px; border-radius: 8px;
                      margin-bottom: 20px; box-shadow: 0 2px 4px rgba(0,0,0,0.1); }
        .info-panel p { margin: 5px 0; }
        .section { background: #fff; border-radius: 8px; margin-bottom: 15px;
                   box-shadow: 0 2px 4px rgba(0,0,0,0.1); overflow: hidden; }
        .section-header { padding: 15px 20px; cursor: pointer; user-select: none;
                         border-bottom: 1px solid #eee; display: flex; justify-content: space-between; align-items: center; }
        .section-header:hover { background: #f8f9fa; }
        .section-header h2 { font-size: 16px; color: #333; }
        .section-header .toggle { color: #999; font-size: 20px; transition: transform 0.2s; }
        .section-content { display: none; padding: 15px 20px; }
        .section.open .section-content { display: block; }
        .section.open .toggle { transform: rotate(90deg); }
        .diff-item { padding: 10px; margin: 5px 0; border-radius: 4px; border-left: 3px solid; }
        .diff-added { background: #e8f5e9; border-color: #4caf50; }
        .diff-removed { background: #ffebee; border-color: #f44336; }
        .diff-modified { background: #fff3e0; border-color: #ff9800; }
        .diff-item-title { font-weight: 600; margin-bottom: 5px; }
        .diff-details { font-size: 13px; color: #666; margin-left: 20px; }
        .diff-details code { background: #f5f5f5; padding: 2px 6px; border-radius: 3px; }
        .type-mapping { color: #666; font-size: 12px; margin-top: 4px; padding-left: 10px; }
        .type-mapping .status-icon { margin-right: 4px; }
        .type-mapping.status-mapped { color: #ff9800; }
        .type-mapping.status-incompatible { color: #f44336; }
        .legend { font-size: 13px; color: #666; margin-top: 10px; }
        .legend span { margin-right: 15px; }
        .summary { display: flex; gap: 15px; flex-wrap: wrap; }
        .summary-card { flex: 1; min-width: 150px; padding: 15px; border-radius: 8px; text-align: center; }
        .summary-card.added { background: #e8f5e9; }
        .summary-card.removed { background: #ffebee; }
        .summary-card.modified { background: #fff3e0; }
        .summary-card h3 { font-size: 24px; margin-bottom: 5px; }
        .summary-card p { color: #666; font-size: 13px; }
        .no-diff { text-align: center; padding: 40px; color: #4caf50; font-size: 18px; }
        table { width: 100%; border-collapse: collapse; margin-top: 10px; }
        th, td { padding: 8px 12px; text-align: left; border-bottom: 1px solid #eee; }
        th { background: #f8f9fa; font-weight: 600; }
    </style>
</head>
<body>
    <div class="container">
        <h1>📊 数据库结构对比报告</h1>

        <div class="info-panel">
            <p><strong>源数据库:</strong> {{.SourceSchema}}</p>
            <p><strong>目标数据库:</strong> {{.TargetSchema}}</p>
            <div class="legend">
                <strong>图例:</strong>
                <span>✅ 相同</span>
                <span>⚠️ 已映射匹配</span>
                <span>❌ 类型不兼容</span>
            </div>
        </div>

        {{if not .HasDiff}}
        <div class="section">
            <div class="no-diff">✓ 两个数据库结构完全一致</div>
        </div>
        {{end}}

        <div class="section open">
            <div class="section-header" onclick="toggleSection(this)">
                <h2>📋 差异摘要</h2>
                <span class="toggle">▶</span>
            </div>
            <div class="section-content">
                <div class="summary">
                    {{if .TableDiff.Added}}
                    <div class="summary-card added">
                        <h3>{{len .TableDiff.Added}}</h3>
                        <p>新增表</p>
                    </div>
                    {{end}}
                    {{if .TableDiff.Missing}}
                    <div class="summary-card removed">
                        <h3>{{len .TableDiff.Missing}}</h3>
                        <p>缺失表</p>
                    </div>
                    {{end}}
                    {{range $table, $diff := .ColumnDiff}}
                        {{if or $diff.Added $diff.Removed $diff.Modified}}
                        <div class="summary-card modified">
                            <h3>{{add (len $diff.Added) (len $diff.Removed) (len $diff.Modified)}}</h3>
                            <p>{{$table}} 列变更</p>
                        </div>
                        {{end}}
                    {{end}}
                </div>
            </div>
        </div>

        {{if or .TableDiff.Added .TableDiff.Missing}}
        <div class="section open">
            <div class="section-header" onclick="toggleSection(this)">
                <h2>📁 表差异 ({{add (len $.TableDiff.Added) (len $.TableDiff.Missing)}})</h2>
                <span class="toggle">▶</span>
            </div>
            <div class="section-content">
                {{range .TableDiff.Added}}
                <div class="diff-item diff-added">
                    <div class="diff-item-title">+ {{.}}</div>
                </div>
                {{end}}
                {{range .TableDiff.Missing}}
                <div class="diff-item diff-removed">
                    <div class="diff-item-title">- {{.}}</div>
                </div>
                {{end}}
            </div>
        </div>
        {{end}}

        {{range $table, $diff := .ColumnDiff}}
        {{if or $diff.Added $diff.Removed $diff.Modified}}
        <div class="section">
            <div class="section-header" onclick="toggleSection(this)">
                <h2>📝 列差异：{{$table}} ({{add (len $diff.Added) (len $diff.Removed) (len $diff.Modified)}})</h2>
                <span class="toggle">▶</span>
            </div>
            <div class="section-content">
                {{range $diff.Added}}
                <div class="diff-item diff-added">
                    <div class="diff-item-title">+ {{.Name}}</div>
                    <div class="diff-details"><code>{{.ColumnType}}</code></div>
                </div>
                {{end}}
                {{range $diff.Removed}}
                <div class="diff-item diff-removed">
                    <div class="diff-item-title">- {{.Name}}</div>
                    <div class="diff-details"><code>{{.ColumnType}}</code></div>
                </div>
                {{end}}
                {{range $diff.Modified}}
                <div class="diff-item diff-modified">
                    <div class="diff-item-title">~ {{.Name}}</div>
                    <div class="diff-details">
                        {{if .TypeMapping}}
                        <div class="type-mapping status-{{.TypeMapping.Status}}">
                            <span class="status-icon">{{typeMappingIcon .TypeMapping.Status}}</span>
                            <span class="status-text">{{typeMappingStatusText .TypeMapping.Status}}</span>
                            {{if eq .TypeMapping.Status "mapped"}}: <code>{{.TypeMapping.MappedType}}</code>{{end}}
                        </div>
                        {{end}}
                        {{range .Changes}}<div>• {{.}}</div>{{end}}
                    </div>
                </div>
                {{end}}
            </div>
        </div>
        {{end}}
        {{end}}

        {{range $table, $diff := .IndexDiff}}
        {{if or $diff.Added $diff.Removed $diff.Modified}}
        <div class="section">
            <div class="section-header" onclick="toggleSection(this)">
                <h2>🔖 索引差异：{{$table}} ({{add (len $diff.Added) (len $diff.Removed) (len $diff.Modified)}})</h2>
                <span class="toggle">▶</span>
            </div>
            <div class="section-content">
                {{range $diff.Added}}
                <div class="diff-item diff-added">
                    <div class="diff-item-title">+ {{.Name}} ({{.IndexType}})</div>
                    <div class="diff-details">列：{{.ColumnName}}</div>
                </div>
                {{end}}
                {{range $diff.Removed}}
                <div class="diff-item diff-removed">
                    <div class="diff-item-title">- {{.Name}} ({{.IndexType}})</div>
                    <div class="diff-details">列：{{.ColumnName}}</div>
                </div>
                {{end}}
                {{range $diff.Modified}}
                <div class="diff-item diff-modified">
                    <div class="diff-item-title">~ {{.Name}}</div>
                    <div class="diff-details">
                        {{range .Changes}}<div>• {{.}}</div>{{end}}
                    </div>
                </div>
                {{end}}
            </div>
        </div>
        {{end}}
        {{end}}

        {{range $table, $diff := .ConstraintDiff}}
        {{if or $diff.Added $diff.Removed $diff.Modified}}
        <div class="section">
            <div class="section-header" onclick="toggleSection(this)">
                <h2>🔒 约束差异：{{$table}} ({{add (len $diff.Added) (len $diff.Removed) (len $diff.Modified)}})</h2>
                <span class="toggle">▶</span>
            </div>
            <div class="section-content">
                {{range $diff.Added}}
                <div class="diff-item diff-added">
                    <div class="diff-item-title">+ {{.Name}} ({{.Type}})</div>
                    <div class="diff-details">列：{{.ColumnName}}</div>
                </div>
                {{end}}
                {{range $diff.Removed}}
                <div class="diff-item diff-removed">
                    <div class="diff-item-title">- {{.Name}} ({{.Type}})</div>
                    <div class="diff-details">列：{{.ColumnName}}</div>
                </div>
                {{end}}
                {{range $diff.Modified}}
                <div class="diff-item diff-modified">
                    <div class="diff-item-title">~ {{.Name}}</div>
                    <div class="diff-details">
                        {{range .Changes}}<div>• {{.}}</div>{{end}}
                    </div>
                </div>
                {{end}}
            </div>
        </div>
        {{end}}
        {{end}}

        {{if .TableCounts}}
        <div class="section open">
            <div class="section-header" onclick="toggleSection(this)">
                <h2>📊 记录数对比</h2>
                <span class="toggle">▶</span>
            </div>
            <div class="section-content">
                <table>
                    <thead>
                        <tr>
                            <th>表名</th>
                            <th>源库记录数</th>
                            <th>目标库记录数</th>
                            <th>差异</th>
                        </tr>
                    </thead>
                    <tbody>
                        {{range $table, $count := .TableCounts}}
                        <tr class="{{if eq $count.Diff 0}}diff-added{{else}}diff-modified{{end}}">
                            <td>{{$table}}</td>
                            <td>{{$count.SourceCount}}</td>
                            <td>{{$count.TargetCount}}</td>
                            <td>{{if gt $count.Diff 0}}+{{$count.Diff}}{{else if lt $count.Diff 0}}{{$count.Diff}}{{else}}={{end}}</td>
                        </tr>
                        {{end}}
                    </tbody>
                </table>
            </div>
        </div>
        {{end}}

        {{range $table, $diff := .TableDataDiff}}
        {{if or $diff.Added $diff.Removed $diff.Modified}}
        <div class="section">
            <div class="section-header" onclick="toggleSection(this)">
                <h2>📄 数据差异：{{$table}}</h2>
                <span class="toggle">▶</span>
            </div>
            <div class="section-content">
                {{if $diff.Added}}
                <div class="diff-item diff-added">
                    <div class="diff-item-title">+ 新增行 ({{len $diff.Added}})</div>
                    <table>
                        <thead><tr><th>主键</th><th>数据</th></tr></thead>
                        <tbody>
                        {{range $i, $row := $diff.Added}}
                        {{if lt $i 10}}
                        <tr><td>{{index $row "id"}}</td><td><pre>{{range $k, $v := $row}}{{$k}}: {{$v}}
{{end}}</pre></td></tr>
                        {{end}}
                        {{end}}
                        {{if gt (len $diff.Added) 10}}
                        <tr><td colspan="2">... 还有 {{sub (len $diff.Added) 10}} 行</td></tr>
                        {{end}}
                        </tbody>
                    </table>
                </div>
                {{end}}
                {{if $diff.Removed}}
                <div class="diff-item diff-removed">
                    <div class="diff-item-title">- 删除行 ({{len $diff.Removed}})</div>
                    <table>
                        <thead><tr><th>主键</th><th>数据</th></tr></thead>
                        <tbody>
                        {{range $i, $row := $diff.Removed}}
                        {{if lt $i 10}}
                        <tr><td>{{index $row "id"}}</td><td><pre>{{range $k, $v := $row}}{{$k}}: {{$v}}
{{end}}</pre></td></tr>
                        {{end}}
                        {{end}}
                        {{if gt (len $diff.Removed) 10}}
                        <tr><td colspan="2">... 还有 {{sub (len $diff.Removed) 10}} 行</td></tr>
                        {{end}}
                        </tbody>
                    </table>
                </div>
                {{end}}
                {{if $diff.Modified}}
                <div class="diff-item diff-modified">
                    <div class="diff-item-title">~ 修改行 ({{len $diff.Modified}})</div>
                    {{range $i, $mod := $diff.Modified}}
                    {{if lt $i 10}}
                    <div class="diff-details">
                        <strong>主键：{{$mod.Key}}</strong><br>
                        {{range $mod.Changes}}• {{.}}<br>{{end}}
                    </div>
                    {{end}}
                    {{end}}
                    {{if gt (len $diff.Modified) 10}}
                    <div class="diff-details">... 还有 {{sub (len $diff.Modified) 10}} 行</div>
                    {{end}}
                </div>
                {{end}}
            </div>
        </div>
        {{end}}
        {{end}}
    </div>

    <script>
        function toggleSection(header) {
            header.parentElement.classList.toggle('open');
        }
    </script>
</body>
</html>
`

	tmpl, err := template.New("report").Funcs(template.FuncMap{
		"add": func(nums ...int) int {
			sum := 0
			for _, n := range nums {
				sum += n
			}
			return sum
		},
		"sub": func(a, b int) int {
			return a - b
		},
		"typeMappingIcon": func(status string) string {
			switch status {
			case "same":
				return "✅"
			case "mapped":
				return "⚠️"
			case "incompatible":
				return "❌"
			default:
				return "?"
			}
		},
		"typeMappingStatusText": func(status string) string {
			switch status {
			case "same":
				return "相同"
			case "mapped":
				return "已映射匹配"
			case "incompatible":
				return "类型不兼容"
			default:
				return "未知"
			}
		},
	}).Parse(tmplContent)
	if err != nil {
		return fmt.Errorf("解析模板失败：%w", err)
	}

	hasDiff := len(result.TableDiff.Added) > 0 || len(result.TableDiff.Missing) > 0
	hasDataDiff := false
	for _, diff := range result.ColumnDiff {
		if len(diff.Added) > 0 || len(diff.Removed) > 0 || len(diff.Modified) > 0 {
			hasDiff = true
			break
		}
	}
	for _, diff := range result.TableDataDiff {
		if len(diff.Added) > 0 || len(diff.Removed) > 0 || len(diff.Modified) > 0 {
			hasDataDiff = true
			break
		}
	}

	data := htmlReportData{
		SourceSchema:   result.SourceSchema,
		TargetSchema:   result.TargetSchema,
		TableDiff:      result.TableDiff,
		ColumnDiff:     result.ColumnDiff,
		IndexDiff:      result.IndexDiff,
		ConstraintDiff: result.ConstraintDiff,
		TableCounts:    result.TableCounts,
		TableDataDiff:  result.TableDataDiff,
		HasDiff:        hasDiff || hasDataDiff,
		HasDataDiff:    hasDataDiff,
	}

	var out *os.File
	if outputFile != "" {
		out, err = os.Create(outputFile)
		if err != nil {
			return fmt.Errorf("创建输出文件失败：%w", err)
		}
		defer out.Close()
	} else {
		out = os.Stdout
	}

	if err := tmpl.Execute(out, data); err != nil {
		return fmt.Errorf("生成报告失败：%w", err)
	}

	return nil
}
