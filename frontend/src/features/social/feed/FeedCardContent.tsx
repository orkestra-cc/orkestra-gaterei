import { Card, Image } from 'react-bootstrap';
import ReactPlayer from 'react-player';
import createMarkup from 'helpers/createMarkup';
import FeedEvent from './FeedEvent';
import classNames from 'classnames';
import FeedUrl from './FeedUrl';
import FeedGallery from 'features/social/feed/FeedGallery';
import FalconLightBox from 'components/common/FalconLightBox';

const FeedCardContent = ({
  status,
  imgSrc,
  gallery,
  feedEvent,
  url,
  video
}) => {
  return (
    <Card.Body className={classNames({ 'p-0': !!feedEvent })}>
      {!!status && <p dangerouslySetInnerHTML={createMarkup(status)} />}
      {!!imgSrc && (
        <FalconLightBox image={imgSrc}>
          <Image src={imgSrc} fluid rounded />
        </FalconLightBox>
      )}
      {!!gallery && <FeedGallery images={gallery} />}
      {!!feedEvent && <FeedEvent {...feedEvent} />}
      {!!url && <FeedUrl {...url} />}
      {!!video && (
        <ReactPlayer
          src={video.src}
          controls={true}
          width="100%"
          height="100%"
          className="react-player"
        />
      )}
    </Card.Body>
  );
};

export default FeedCardContent;
