import { Router } from 'express';
import { EventsController } from '../controllers/events.controller';

const eventsRoutes = Router();
const eventsController = new EventsController();

eventsRoutes.get('/events', eventsController.getAll);
eventsRoutes.get('/events/:id', eventsController.getById);

export { eventsRoutes }; 