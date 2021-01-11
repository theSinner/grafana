import { LegendOptions, GraphTooltipOptions } from '@grafana/ui';

export interface XYDimensionConfig {
  frame: number;
  sort?: boolean;
  x?: string; // name | first
  exclude?: string[]; // all other numbers except
}

export interface Options {
  dims: XYDimensionConfig;
  legend: LegendOptions;
  tooltipOptions: GraphTooltipOptions;
}
