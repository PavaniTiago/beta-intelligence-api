import { Request, Response } from 'express';
import { GetEventsUseCase } from '../use-cases/events/get-events.usecase';
import { EventsRepository } from '../repositories/events.repository';

export class EventsController {
  async getAll(req: Request, res: Response) {
    try {
      const eventsRepository = new EventsRepository();
      const getEventsUseCase = new GetEventsUseCase(eventsRepository);

      const events = await getEventsUseCase.execute();

      return res.json(events);
    } catch (error) {
      return res.status(500).json({ error: 'Internal server error' });
    }
  }

  async getById(req: Request, res: Response) {
    try {
      const { id } = req.params;
      const eventsRepository = new EventsRepository();
      const getEventsUseCase = new GetEventsUseCase(eventsRepository);

      const event = await getEventsUseCase.getById(id);

      if (!event) {
        return res.status(404).json({ error: 'Event not found' });
      }

      return res.json(event);
    } catch (error) {
      return res.status(500).json({ error: 'Internal server error' });
    }
  }
} 