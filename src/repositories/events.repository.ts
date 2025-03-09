import { prisma } from '../lib/prisma';
import { Event } from '../types/events';

export class EventsRepository {
  async findAll(): Promise<Event[]> {
    const events = await prisma.events.findMany({
      include: {
        user: {
          select: {
            phone: true,
            email: true,
            isClient: true,
            fullName: true
          }
        }
      }
    });

    return events;
  }

  async findById(event_id: string): Promise<Event | null> {
    const event = await prisma.events.findUnique({
      where: { event_id },
      include: {
        user: {
          select: {
            phone: true,
            email: true,
            isClient: true,
            fullName: true
          }
        }
      }
    });

    return event;
  }
} 