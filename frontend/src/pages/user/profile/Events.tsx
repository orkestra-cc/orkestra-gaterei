

import FalconCardHeader from 'components/common/FalconCardHeader';
import { Card } from 'react-bootstrap';
import FalconCardFooterLink from 'components/common/FalconCardFooterLink';
import Event from 'features/events/event-list/Event';
import paths from 'routes/paths';

const Events = ({ cardTitle, events, ...rest }) => {
  return (
    <Card {...rest}>
      <FalconCardHeader title={cardTitle} light />
      <Card.Body className="fs-10 border-bottom">
        {events.map((event, index) => (
          <Event
            key={event.id}
            details={event}
            isLast={index === events.length - 1}
          />
        ))}
      </Card.Body>
      <FalconCardFooterLink title="All Events" to={paths.eventList} size="sm" />
    </Card>
  );
};

export default Events;
