export interface Event {
  event_id: string;
  event_name: string;
  pageview_id: string;
  session_id: string;
  event_time: Date;
  user_id: string;
  profession_id: number;
  product_id: number;
  funnel_id: number;
  event_source: string;
  event_type: string;
  user?: {
    phone: string;
    email: string;
    isClient: boolean;
    fullName: string;
  }
} 