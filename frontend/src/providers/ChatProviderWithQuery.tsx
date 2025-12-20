import React, { useState, createContext, useContext, ReactNode } from 'react';
import {
  useChatThreads,
  useFlattenedChatMessages,
  useChatContacts,
  useChatGroups,
  useSendMessage,
  useCreateThread,
  useMarkThreadAsRead
} from 'hooks/useChat';

// Type definitions
interface ChatThread {
  id: number;
  type: 'user' | 'group';
  userId?: number;
  groupId?: number;
  unreadCount: number;
  lastMessage?: string;
  timestamp?: string;
}

interface ChatContact {
  id: number;
  name: string;
  avatarSrc: string;
  status?: string;
  email?: string;
}

interface ChatGroup {
  id: number;
  name: string;
  members: Array<{ id: number; avatarSrc: string }>;
}

interface ChatMessage {
  id: number;
  threadId: number;
  senderId: number;
  content: string;
  timestamp: string;
  attachments?: any[];
}

interface ChatUser {
  name: string;
  avatarSrc: string | string[];
}

interface ChatContextValue {
  // Data
  threads: ChatThread[];
  contacts: ChatContact[];
  groups: ChatGroup[];
  messages: ChatMessage[];
  currentThread: ChatThread | null;
  
  // Loading states
  isLoadingThreads: boolean;
  isLoadingMessages: boolean;
  isLoadingContacts: boolean;
  isLoadingGroups: boolean;
  
  // Error states
  threadsError: Error | null;
  messagesError: Error | null;
  contactsError: Error | null;
  groupsError: Error | null;
  
  // Mutation states
  isSendingMessage: boolean;
  isCreatingThread: boolean;
  
  // Helper functions
  getUser: (thread: ChatThread | null) => ChatUser | ChatContact;
  
  // Actions
  sendMessage: (message: string, attachments?: any[]) => Promise<void>;
  createThread: (participants: number[], isGroup?: boolean, groupName?: string) => Promise<void>;
  selectThread: (thread: ChatThread) => void;
  markThreadAsRead: (threadId: number) => Promise<void>;
  
  // UI State
  textAreaInitialHeight: number;
  setTextAreaInitialHeight: (height: number) => void;
  isOpenThreadInfo: boolean;
  setIsOpenThreadInfo: (open: boolean) => void;
  scrollToBottom: boolean;
  setScrollToBottom: (scroll: boolean) => void;
  
  // Manual refetch functions
  refetchThreads: () => void;
  refetchMessages: () => void;
  refetchContacts: () => void;
  refetchGroups: () => void;
  
  // Infinite loading for messages
  hasNextPage?: boolean;
  fetchNextPage: () => void;
  isFetchingNextPage: boolean;
}

interface ChatProviderWithQueryProps {
  children: ReactNode;
}

export const ChatContext = createContext<ChatContextValue | undefined>(undefined);

const ChatProviderWithQuery: React.FC<ChatProviderWithQueryProps> = ({ children }) => {
  // Local UI state
  const [currentThread, setCurrentThread] = useState<ChatThread | null>(null);
  const [textAreaInitialHeight, setTextAreaInitialHeight] = useState<number>(32);
  const [isOpenThreadInfo, setIsOpenThreadInfo] = useState<boolean>(false);
  const [scrollToBottom, setScrollToBottom] = useState<boolean>(true);

  // TanStack Query for data fetching
  const threadsQuery = useChatThreads();
  const contactsQuery = useChatContacts();
  const groupsQuery = useChatGroups();
  
  // Messages for the current thread
  const messagesQuery = useFlattenedChatMessages(currentThread?.id);
  
  // Mutations
  const sendMessageMutation = useSendMessage({
    onSuccess: () => {
      setScrollToBottom(true);
    },
  });
  
  const createThreadMutation = useCreateThread({
    onSuccess: (newThread) => {
      setCurrentThread(newThread);
      setScrollToBottom(true);
    },
  });
  
  const markAsReadMutation = useMarkThreadAsRead();

  // Helper functions
  const getUser = (thread: ChatThread | null): ChatUser | ChatContact => {
    if (!thread) return { name: '', avatarSrc: '' };
    
    if (thread.type === 'group') {
      const group = groupsQuery.data?.find(g => g.id === thread.groupId);
      if (group) {
        return {
          name: group.name,
          avatarSrc: group.members?.map(member => member.avatarSrc) || []
        };
      }
    } else {
      const contact = contactsQuery.data?.find(c => c.id === thread.userId);
      return contact || { id: 0, name: '', avatarSrc: '' };
    }
    return { name: '', avatarSrc: '' };
  };

  const sendMessage = async (message: string, attachments: any[] = []) => {
    if (!currentThread) return;
    
    try {
      await sendMessageMutation.mutateAsync({
        threadId: currentThread.id,
        message,
        attachments
      });
    } catch (error) {
      console.error('Failed to send message:', error);
    }
  };

  const createThread = async (participants: number[], isGroup = false, groupName = '') => {
    try {
      await createThreadMutation.mutateAsync({
        participants,
        isGroup,
        groupName
      });
    } catch (error) {
      console.error('Failed to create thread:', error);
    }
  };

  const markThreadAsRead = async (threadId: number) => {
    try {
      await markAsReadMutation.mutateAsync(threadId);
    } catch (error) {
      console.error('Failed to mark thread as read:', error);
    }
  };

  const selectThread = (thread: ChatThread) => {
    setCurrentThread(thread);
    setIsOpenThreadInfo(false);
    setScrollToBottom(true);
    
    // Mark as read if it has unread messages
    if (thread && thread.unreadCount > 0) {
      markThreadAsRead(thread.id);
    }
  };

  const value: ChatContextValue = {
    // Data
    threads: threadsQuery.data?.threads || [],
    contacts: contactsQuery.data || [],
    groups: groupsQuery.data || [],
    messages: messagesQuery.messages || [],
    currentThread,
    
    // Loading states
    isLoadingThreads: threadsQuery.isLoading,
    isLoadingMessages: messagesQuery.isLoading,
    isLoadingContacts: contactsQuery.isLoading,
    isLoadingGroups: groupsQuery.isLoading,
    
    // Error states
    threadsError: threadsQuery.error,
    messagesError: messagesQuery.error,
    contactsError: contactsQuery.error,
    groupsError: groupsQuery.error,
    
    // Mutation states
    isSendingMessage: sendMessageMutation.isPending,
    isCreatingThread: createThreadMutation.isPending,
    
    // Helper functions
    getUser,
    
    // Actions
    sendMessage,
    createThread,
    selectThread,
    markThreadAsRead,
    
    // UI State
    textAreaInitialHeight,
    setTextAreaInitialHeight,
    isOpenThreadInfo,
    setIsOpenThreadInfo,
    scrollToBottom,
    setScrollToBottom,
    
    // Manual refetch functions
    refetchThreads: threadsQuery.refetch,
    refetchMessages: messagesQuery.refetch,
    refetchContacts: contactsQuery.refetch,
    refetchGroups: groupsQuery.refetch,
    
    // Infinite loading for messages
    hasNextPage: messagesQuery.hasNextPage,
    fetchNextPage: messagesQuery.fetchNextPage,
    isFetchingNextPage: messagesQuery.isFetchingNextPage
  };

  return (
    <ChatContext.Provider value={value}>
      {children}
    </ChatContext.Provider>
  );
};

export const useChatContext = (): ChatContextValue => {
  const context = useContext(ChatContext);
  if (!context) {
    throw new Error('useChatContext must be used within ChatProviderWithQuery');
  }
  return context;
};

export default ChatProviderWithQuery;