export interface QueueMetricsItem {
  itemId: string;
  title: string;
  waiting: number;
  granted: number;
  stockLeft: number;
}

export interface QueueMetrics {
  totalWaiting: number;
  totalGranted: number;
  purchased: number;
  expired: number;
  conversion: number | null;
  avgWaitSeconds: number | null;
  items: QueueMetricsItem[];
}
