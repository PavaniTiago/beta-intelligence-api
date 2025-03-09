import { Event } from '../../types/events';
import { EventsRepository } from '../../repositories/events.repository';

export class GetEventsUseCase {
  constructor(private eventsRepository: EventsRepository) {}

  async execute(): Promise<Event[]> {
    const events = await this.eventsRepository.findAll();
    return events;
  }

  async getById(event_id: string): Promise<Event | null> {
    const event = await this.eventsRepository.findById(event_id);
    return event;
  }
} 