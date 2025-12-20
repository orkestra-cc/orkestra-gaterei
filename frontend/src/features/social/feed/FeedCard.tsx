
import FeedCardHeader from './FeedCardHeader';
import { Card } from 'react-bootstrap';
import FeedCardContent from './FeedCardContent';
import FeedCardFooter from './FeedCardFooter';

// Type definitions for Social features
interface SocialUser {
  id: string;
  name: string;
  avatarSrc: string;
  username?: string;
  isOnline?: boolean;
  lastActivity?: string;
}

interface SocialFeedContent {
  text?: string;
  images?: string[];
  video?: string;
  link?: SocialFeedLink;
  poll?: SocialPoll;
}

interface SocialFeedLink {
  url: string;
  title: string;
  description: string;
  image?: string;
}

interface SocialPoll {
  question: string;
  options: SocialPollOption[];
  totalVotes: number;
  expiresAt?: string;
}

interface SocialPollOption {
  id: string;
  text: string;
  votes: number;
  percentage: number;
}

interface SocialFeedDetails {
  likes: number;
  comments: number;
  shares: number;
  likedByCurrentUser?: boolean;
  timestamp: string;
}

interface SocialFeed {
  id: string;
  user: SocialUser;
  content: SocialFeedContent;
  details: SocialFeedDetails;
  type: 'post' | 'share' | 'event' | 'poll';
}

interface FeedCardProps {
  feed: SocialFeed;
  [key: string]: any;
}

const FeedCard: React.FC<FeedCardProps> = ({ feed, ...rest }) => {
  const { id, user, content, details } = feed;
  return (
    <Card {...rest}>
      {!!user && <FeedCardHeader {...user} />}
      {!!content && <FeedCardContent {...content} />}
      {!!details && <FeedCardFooter id={id} {...details} />}
    </Card>
  );
};

FeedCard.Header = FeedCardHeader;

export default FeedCard;
