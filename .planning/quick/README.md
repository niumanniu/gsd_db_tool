# Quick Task: 平均进度显示

## Task
将进度显示改为平均进度模式 `(source+target)/2`

## Changes
- 修改 `pkg/comparator/comparator.go` 中两处进度显示代码
- 计算平均进度 `avgProgress = (sourceProgress + targetProgress) / 2`
- 显示格式：`进度：XX.X% (源：XX.X% | 目标：XX.X%)`

## Before
```
[core_license_photo_en] Source: 24.0% (2000/8324) | Target: 24.0% (2000/8323), 已耗时 2.29s, 本批：串行预估 2238ms, 并行实际 1119ms, 节省 50.0%
```

## After
```
[core_license_photo_en] 进度：24.0% (源：24.0% | 目标：24.0%), 已耗时 2.29s, 本批：1119ms (节省 50.0%)
```
