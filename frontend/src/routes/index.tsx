import React, { Suspense, lazy } from 'react';
import App from 'App';
import paths, { rootPaths } from './paths';
import { Navigate, createBrowserRouter, RouteObject } from 'react-router';

import MainLayout from '../layouts/MainLayout';
import ErrorLayout from '../layouts/ErrorLayout';
import Landing from 'pages/landing/Landing';
import ProtectedRoute from 'components/authentication/ProtectedRoute';
const Accordion = lazy(() => import('docs/components/Accordion'));
const Alerts = lazy(() => import('docs/components/Alerts'));
const Badges = lazy(() => import('docs/components/Badges'));
const Breadcrumbs = lazy(() => import('docs/components/Breadcrumb'));
const Buttons = lazy(() => import('docs/components/Buttons'));
const CalendarExample = lazy(() => import('docs/components/CalendarExample'));
const Cards = lazy(() => import('docs/components/Cards'));
const Dropdowns = lazy(() => import('docs/components/Dropdowns'));
const ListGroups = lazy(() => import('docs/components/ListGroups'));
const Modals = lazy(() => import('docs/components/Modals'));
const Offcanvas = lazy(() => import('docs/components/Offcanvas'));
const Pagination = lazy(() => import('docs/components/Pagination'));
const BasicProgressBar = lazy(() => import('docs/components/ProgressBar'));
const Spinners = lazy(() => import('docs/components/Spinners'));
const Toasts = lazy(() => import('docs/components/Toasts'));
const Avatar = lazy(() => import('docs/components/Avatar'));
const Image = lazy(() => import('docs/components/Image'));
const Tooltips = lazy(() => import('docs/components/Tooltips'));
const Popovers = lazy(() => import('docs/components/Popovers'));
const Figures = lazy(() => import('docs/components/Figures'));
const Hoverbox = lazy(() => import('docs/components/Hoverbox'));
const Tables = lazy(() => import('docs/components/Tables'));
const FormControl = lazy(() => import('docs/components/FormControl'));
const InputGroup = lazy(() => import('docs/components/InputGroup'));
const Select = lazy(() => import('docs/components/Select'));
const Checks = lazy(() => import('docs/components/Checks'));
const Range = lazy(() => import('docs/components/Range'));
const FormLayout = lazy(() => import('docs/components/FormLayout'));
const FloatingLabels = lazy(() => import('docs/components/FloatingLabels'));
const FormValidation = lazy(() => import('docs/components/FormValidation'));
const BootstrapCarousel = lazy(
  () => import('docs/components/BootstrapCarousel')
);
const SlickCarousel = lazy(() => import('docs/components/SlickCarousel'));
const Navs = lazy(() => import('docs/components/Navs'));
const Navbars = lazy(() => import('docs/components/Navbars'));
const Tabs = lazy(() => import('docs/components/Tabs'));
const Collapse = lazy(() => import('docs/components/Collapse'));
const CountUp = lazy(() => import('docs/components/CountUp'));
const Embed = lazy(() => import('docs/components/Embed'));
const Backgrounds = lazy(() => import('docs/components/Backgrounds'));
const Search = lazy(() => import('docs/components/Search'));
const VerticalNavbar = lazy(() => import('docs/components/VerticalNavbar'));
const NavBarTop = lazy(() => import('docs/components/NavBarTop'));
const NavbarDoubleTop = lazy(() => import('docs/components/NavbarDoubleTop'));
const ComboNavbar = lazy(() => import('docs/components/ComboNavbar'));
const TypedText = lazy(() => import('docs/components/TypedText'));
const FileUploader = lazy(() => import('docs/components/FileUploader'));
const Borders = lazy(() => import('docs/utilities/Borders'));
const Colors = lazy(() => import('docs/utilities/Colors'));
const Background = lazy(() => import('docs/utilities/Background'));
const ColoredLinks = lazy(() => import('docs/utilities/ColoredLinks'));
const Display = lazy(() => import('docs/utilities/Display'));
const Visibility = lazy(() => import('docs/utilities/Visibility'));
const StretchedLink = lazy(() => import('docs/utilities/StretchedLink'));
const Float = lazy(() => import('docs/utilities/Float'));
const Position = lazy(() => import('docs/utilities/Position'));
const Spacing = lazy(() => import('docs/utilities/Spacing'));
const Sizing = lazy(() => import('docs/utilities/Sizing'));
const TextTruncation = lazy(() => import('docs/utilities/TextTruncation'));
const Typography = lazy(() => import('docs/utilities/Typography'));
const VerticalAlign = lazy(() => import('docs/utilities/VerticalAlign'));
const Flex = lazy(() => import('docs/utilities/Flex'));
const Grid = lazy(() => import('docs/utilities/Grid'));
const WizardForms = lazy(() => import('docs/components/WizardForms'));
const GettingStarted = lazy(() => import('docs/documentation/GettingStarted'));
const Configuration = lazy(() => import('docs/documentation/Configuration'));
const DarkMode = lazy(() => import('docs/documentation/DarkMode'));
const Plugins = lazy(() => import('docs/documentation/Plugins'));
const Styling = lazy(() => import('docs/documentation/Styling'));
const DesignFile = lazy(() => import('docs/documentation/DesignFile'));
const Starter = lazy(() => import('pages/Starter'));
const AnimatedIcons = lazy(() => import('docs/components/AnimatedIcons'));
const DatePicker = lazy(() => import('docs/components/DatePicker'));
const FontAwesome = lazy(() => import('docs/components/FontAwesome'));
const Changelog = lazy(() => import('docs/documentation/change-log/ChangeLog'));
const Analytics = lazy(() => import('demos/dashboards/AnalyticsDashboard'));
const Crm = lazy(() => import('demos/dashboards/CrmDashboard'));
const Saas = lazy(() => import('demos/dashboards/SaasDashboard'));
const Profile = lazy(() => import('pages/operatore/profile/OperatoreProfile'));
const Associations = lazy(() => import('pages/asscociations/Associations'));
const Followers = lazy(() => import('features/social/followers/Followers'));
const Notifications = lazy(
  () => import('features/social/notifications/Notifications')
);
const ActivityLog = lazy(
  () => import('features/social/activity-log/ActivityLog')
);
const Settings = lazy(() => import('pages/user/settings/Settings'));
const Feed = lazy(() => import('features/social/feed/Feed'));
const Placeholder = lazy(() => import('docs/components/Placeholder'));
const Lightbox = lazy(() => import('docs/components/Lightbox'));
const AdvanceTableExamples = lazy(
  () => import('docs/components/AdvanceTableExamples')
);
const Calendar = lazy(() => import('features/calendar/Calendar'));
import FaqAlt from 'pages/faq/faq-alt/FaqAlt';
import FaqBasic from 'pages/faq/faq-basic/FaqBasic';
import FaqAccordion from 'pages/faq/faq-accordion/FaqAccordion';
import PrivacyPolicy from 'pages/miscellaneous/privacy-policy/PrivacyPolicy';
import InvitePeople from 'pages/miscellaneous/invite-people/InvitePeople';
import AuthTestPage from 'pages/test/AuthTestPage';
import RoleNavigationTester from 'pages/test/RoleNavigationTester';
import PricingDefault from 'pages/pricing/pricing-default/PricingDefault';
import PricingAlt from 'pages/pricing/pricing-alt/PricingAlt';
import CreateEvent from 'features/events/create-an-event/CreateEvent';
import EventList from 'features/events/event-list/EventList';
import EventDetail from 'features/events/event-detail/EventDetail';
import EmailDetail from 'features/email/email-detail/EmailDetail';
import Compose from 'features/email/compose/Compose';
import Inbox from 'features/email/inbox/Inbox';
import Rating from 'docs/components/Rating';
import AdvanceSelect from 'docs/components/AdvanceSelect';
import Editor from 'docs/components/Editor';
import Chat from 'features/chat/Chat';
const Kanban = lazy(() => import('features/kanban/Kanban'));
import DraggableExample from 'docs/components/DraggableExample';
const LineCharts = lazy(
  () => import('docs/components/charts-example/echarts/line-charts')
);
const BarCharts = lazy(
  () => import('docs/components/charts-example/echarts/bar-charts')
);
const CandlestickCharts = lazy(
  () => import('docs/components/charts-example/echarts/candlestick-charts')
);
const GeoMaps = lazy(
  () => import('docs/components/charts-example/echarts/geo-map')
);
const ScatterCharts = lazy(
  () => import('docs/components/charts-example/echarts/scatter-charts')
);
const PieCharts = lazy(
  () => import('docs/components/charts-example/echarts/pie-charts')
);
const RadarCharts = lazy(
  () => import('docs/components/charts-example/echarts/radar-charts/Index')
);
const HeatmapCharts = lazy(
  () => import('docs/components/charts-example/echarts/heatmap-chart')
);
const Chartjs = lazy(() => import('docs/components/charts-example/chartjs'));
const D3js = lazy(() => import('docs/components/charts-example/d3'));
import HowToUse from 'docs/components/charts-example/echarts/HowToUse';
const GoogleMapExample = lazy(() => import('docs/components/GoogleMapExample'));
import LeafletMapExample from 'docs/components/LeafletMapExample';
import CookieNoticeExample from 'docs/components/CookieNoticeExample';
import Scrollbar from 'docs/components/Scrollbar';
import Scrollspy from 'docs/components/Scrollspy';
import ReactIcons from 'docs/components/ReactIcons';
import ReactPlayerExample from 'docs/components/ReactPlayerExample';
import EmojiPickerExample from 'docs/components/EmojiPicker';
import TreeviewExample from 'docs/components/TreeviewExample';
import Timeline from 'docs/components/Timeline';
const Widgets = lazy(() => import('widgets/Widgets'));
const ProjectManagement = lazy(
  () => import('demos/dashboards/ProjectManagementDashboard')
);
const Migration = lazy(() => import('docs/documentation/migration/Migration'));

import Error401 from 'components/errors/Error401';
import Error404 from 'components/errors/Error404';
import Error500 from 'components/errors/Error500';

import Login from 'components/authentication/Login';
import SocialAuthCallback from 'components/authentication/SocialAuthCallback';
const Dashboard = lazy(() => import('demos/dashboards/DefaultDashboard'));
import Faq from 'docs/documentation/Faq';
const SupportDesk = lazy(() => import('demos/dashboards/SupportDeskDashboard'));
const UserManagement = lazy(
  () => import('demos/dashboards/UserManagementDashboard')
);
const AdminUserProfile = lazy(
  () => import('pages/admin/user-profile/AdminUserProfile')
);
const DeadlineReports = lazy(
  () => import('pages/admin/Reports/DeadlineReports')
);
const OperatoreProfile = lazy(
  () => import('pages/operatore/profile/OperatoreProfile')
);
const VehicleManagement = lazy(
  () => import('demos/dashboards/VehicleManagementDashboard')
);
const VehicleProfile = lazy(
  () => import('pages/fleet/vehicle-profile/VehicleProfile')
);
const CraneManagement = lazy(
  () => import('demos/dashboards/CraneManagementDashboard')
);
const CraneProfile = lazy(
  () => import('pages/fleet/crane-profile/CraneProfile')
);
const TachographManagement = lazy(
  () => import('demos/dashboards/TachographManagementDashboard')
);
const TachographProfile = lazy(
  () => import('pages/fleet/tachograph-profile/TachographProfile')
);
import TableView from 'features/support-desk/tickets-layout/TableView';
import CardView from 'features/support-desk/tickets-layout/CardView';
import Contacts from 'features/support-desk/contacts/Contacts';
import ContactDetails from 'features/support-desk/contact-details/ContactDetails';
import TicketsPreview from 'features/support-desk/tickets-preview/TicketsPreview';
import QuickLinks from 'features/support-desk/quick-links/QuickLinks';
import Reports from 'features/support-desk/reports/Reports';
import InputMaskExample from 'docs/components/InputMaskExample';
import RangeSlider from 'docs/components/RangeSlider';
import VerticalNavLayout from 'layouts/VerticalNavLayout';
import TopNavLayout from 'layouts/TopNavLayout';
import ComboNavLayout from 'layouts/ComboNavLayout';
import DoubleTopNavLayout from 'layouts/DoubleTopNavLayout';
import FalconLoader from 'components/common/FalconLoader';

const routes: RouteObject[] = [
  {
    element: <App />,
    children: [
      {
        path: 'landing',
        element: <Landing />
      },
      {
        path: rootPaths.errorsRoot,
        element: <ErrorLayout />,
        children: [
          {
            path: paths.error401,
            element: <Error401 />
          },
          {
            path: paths.error404,
            element: <Error404 />
          },
          {
            path: paths.error500,
            element: <Error500 />
          }
        ]
      },
      {
        path: 'login',
        element: <Login />
      },
      {
        path: 'auth/callback',
        element: <SocialAuthCallback />
      },
      {
        path: '/',
        element: (
          <ProtectedRoute>
            <MainLayout />
          </ProtectedRoute>
        ),
        children: [
          {
            index: true,
            element: <Navigate to="/dashboard/analytics" replace />
          },
          {
            path: rootPaths.dashboardRoot,
            children: [
              {
                path: paths.analytics,
                element: (
                  <Suspense
                    key="dashboard-analytics"
                    fallback={<FalconLoader />}
                  >
                    <Analytics />
                  </Suspense>
                )
              },
              {
                path: paths.crm,
                element: (
                  <Suspense key="dashboard-crm" fallback={<FalconLoader />}>
                    <Crm />
                  </Suspense>
                )
              },
              {
                path: paths.saas,
                element: (
                  <Suspense key="dashboard-sass" fallback={<FalconLoader />}>
                    <Saas />
                  </Suspense>
                )
              },
              {
                path: paths.projectManagement,
                element: (
                  <Suspense
                    key="dashboard-projectManagement"
                    fallback={<FalconLoader />}
                  >
                    <ProjectManagement />
                  </Suspense>
                )
              },
              {
                path: paths.supportDesk,
                element: (
                  <Suspense
                    key="dashboard-supportDesk"
                    fallback={<FalconLoader />}
                  >
                    <SupportDesk />
                  </Suspense>
                )
              }
            ]
          },
          {
            path: rootPaths.appsRoot,
            children: [
              {
                path: paths.calendar,
                element: (
                  <Suspense key="calendar" fallback={<FalconLoader />}>
                    <Calendar />
                  </Suspense>
                )
              },
              {
                path: paths.chat,
                element: <Chat />
              },
              {
                path: paths.kanban,
                element: (
                  <Suspense key="kanban" fallback={<FalconLoader />}>
                    <Kanban />
                  </Suspense>
                )
              }
            ]
          },
          {
            path: rootPaths.emailRoot,
            children: [
              {
                path: paths.emailInbox,
                element: <Inbox />
              },
              {
                path: paths.emailDetail,
                element: <EmailDetail />
              },
              {
                path: paths.emailCompose,
                element: <Compose />
              }
            ]
          },
          {
            path: rootPaths.eventsRoot,
            children: [
              {
                path: paths.createEvent,
                element: <CreateEvent />
              },
              {
                path: paths.eventDetail,
                element: <EventDetail />
              },
              {
                path: paths.eventList,
                element: <EventList />
              }
            ]
          },
          {
            path: rootPaths.socialRoot,
            children: [
              {
                path: paths.feed,
                element: <Feed />
              },
              {
                path: paths.activityLog,
                element: <ActivityLog />
              },
              {
                path: paths.notifications,
                element: <Notifications />
              },
              {
                path: paths.followers,
                element: <Followers />
              }
            ]
          },
          {
            path: rootPaths.supportDeskRoot,
            children: [
              {
                path: paths.ticketsTable,
                element: <TableView />
              },
              {
                path: paths.ticketsCard,
                element: <CardView />
              },
              {
                path: paths.contacts,
                element: <Contacts />
              },
              {
                path: paths.contactDetails,
                element: <ContactDetails />
              },
              {
                path: paths.ticketsPreview,
                element: <TicketsPreview />
              },
              {
                path: paths.quickLinks,
                element: <QuickLinks />
              },
              {
                path: paths.reports,
                element: <Reports />
              }
            ]
          },
          {
            path: rootPaths.pagesRoot,
            children: [
              {
                path: paths.starter,
                element: <Starter />
              }
            ]
          },
          {
            path: rootPaths.userRoot,
            children: [
              {
                path: paths.userProfile,
                element: <Profile />
              },
              {
                path: paths.userSettings,
                element: <Settings />
              }
            ]
          },
          {
            path: rootPaths.pricingRoot,
            children: [
              {
                path: paths.pricingDefault,
                element: <PricingDefault />
              },
              {
                path: paths.pricingAlt,
                element: <PricingAlt />
              }
            ]
          },
          {
            path: rootPaths.faqRoot,
            children: [
              {
                path: paths.faqBasic,
                element: <FaqBasic />
              },
              {
                path: paths.faqAlt,
                element: <FaqAlt />
              },
              {
                path: paths.faqAccordion,
                element: <FaqAccordion />
              }
            ]
          },
          {
            path: rootPaths.miscRoot,
            children: [
              {
                path: paths.associations,
                element: <Associations />
              },
              {
                path: paths.invitePeople,
                element: <InvitePeople />
              },
              {
                path: paths.privacyPolicy,
                element: <PrivacyPolicy />
              },
              {
                path: paths.authTest,
                element: <AuthTestPage />
              }
            ]
          },
          {
            path: rootPaths.adminRoot,
            children: [
              {
                path: paths.userManagement,
                element: (
                  <ProtectedRoute
                    requiredPermissions={[
                      ['developer', 'ceo', 'administrator']
                    ]}
                  >
                    <Suspense
                      key="admin-userManagement"
                      fallback={<FalconLoader />}
                    >
                      <UserManagement />
                    </Suspense>
                  </ProtectedRoute>
                )
              },
              {
                path: 'user/profile/:userId',
                element: (
                  <ProtectedRoute
                    requiredPermissions={[
                      ['developer', 'ceo', 'administrator']
                    ]}
                  >
                    <Suspense
                      key="admin-userProfile"
                      fallback={<FalconLoader />}
                    >
                      <AdminUserProfile />
                    </Suspense>
                  </ProtectedRoute>
                )
              },
              {
                path: 'reports/deadlines',
                element: (
                  <ProtectedRoute
                    requiredPermissions={[
                      ['developer', 'ceo', 'administrator', 'manager']
                    ]}
                  >
                    <Suspense
                      key="admin-deadlineReports"
                      fallback={<FalconLoader />}
                    >
                      <DeadlineReports />
                    </Suspense>
                  </ProtectedRoute>
                )
              }
            ]
          },
          {
            path: rootPaths.userRoot,
            children: [
              {
                path: 'profile',
                element: (
                  <Suspense key="operatore-profile" fallback={<FalconLoader />}>
                    <OperatoreProfile />
                  </Suspense>
                )
              }
            ]
          },
          {
            path: 'fleet/vehicles',
            element: (
              <ProtectedRoute
                requiredPermissions={[
                  ['developer', 'ceo', 'administrator']
                ]}
              >
                <Suspense
                  key="fleet-vehicleManagement"
                  fallback={<FalconLoader />}
                >
                  <VehicleManagement />
                </Suspense>
              </ProtectedRoute>
            )
          },
          {
            path: 'fleet/vehicle/:vehicleId',
            element: (
              <ProtectedRoute
                requiredPermissions={[
                  ['developer', 'ceo', 'administrator']
                ]}
              >
                <Suspense
                  key="fleet-vehicleProfile"
                  fallback={<FalconLoader />}
                >
                  <VehicleProfile />
                </Suspense>
              </ProtectedRoute>
            )
          },
          {
            path: 'fleet/cranes',
            element: (
              <ProtectedRoute
                requiredPermissions={[
                  ['developer', 'ceo', 'administrator']
                ]}
              >
                <Suspense
                  key="fleet-craneManagement"
                  fallback={<FalconLoader />}
                >
                  <CraneManagement />
                </Suspense>
              </ProtectedRoute>
            )
          },
          {
            path: 'fleet/crane/:craneId',
            element: (
              <ProtectedRoute
                requiredPermissions={[
                  ['developer', 'ceo', 'administrator']
                ]}
              >
                <Suspense key="fleet-craneProfile" fallback={<FalconLoader />}>
                  <CraneProfile />
                </Suspense>
              </ProtectedRoute>
            )
          },
          {
            path: 'fleet/tachographs',
            element: (
              <ProtectedRoute
                requiredPermissions={[
                  ['developer', 'ceo', 'administrator']
                ]}
              >
                <Suspense
                  key="fleet-tachographManagement"
                  fallback={<FalconLoader />}
                >
                  <TachographManagement />
                </Suspense>
              </ProtectedRoute>
            )
          },
          {
            path: 'fleet/tachograph/:tachographId',
            element: (
              <ProtectedRoute
                requiredPermissions={[
                  ['developer', 'ceo', 'administrator']
                ]}
              >
                <Suspense
                  key="fleet-tachographProfile"
                  fallback={<FalconLoader />}
                >
                  <TachographProfile />
                </Suspense>
              </ProtectedRoute>
            )
          },
          {
            path: rootPaths.formsRoot,
            children: [
              {
                path: rootPaths.basicFormsRoot,
                children: [
                  {
                    path: paths.formControl,
                    element: <FormControl />
                  },
                  {
                    path: paths.inputGroup,
                    element: <InputGroup />
                  },
                  {
                    path: paths.select,
                    element: <Select />
                  },
                  {
                    path: paths.checks,
                    element: <Checks />
                  },
                  {
                    path: paths.range,
                    element: <Range />
                  },
                  {
                    path: paths.formLayout,
                    element: <FormLayout />
                  }
                ]
              },
              {
                path: rootPaths.advanceFormsRoot,
                children: [
                  {
                    path: paths.advanceSelect,
                    element: <AdvanceSelect />
                  },
                  {
                    path: paths.datePicker,
                    element: <DatePicker />
                  },
                  {
                    path: paths.editor,
                    element: <Editor />
                  },
                  {
                    path: paths.emojiButton,
                    element: <EmojiPickerExample />
                  },
                  {
                    path: paths.fileUploader,
                    element: <FileUploader />
                  },
                  {
                    path: paths.inputMask,
                    element: <InputMaskExample />
                  },
                  {
                    path: paths.rangeSlider,
                    element: <RangeSlider />
                  },
                  {
                    path: paths.rating,
                    element: <Rating />
                  }
                ]
              },
              {
                path: paths.floatingLabels,
                element: <FloatingLabels />
              },
              {
                path: paths.wizard,
                element: <WizardForms />
              },
              {
                path: paths.validation,
                element: <FormValidation />
              }
            ]
          },
          {
            path: rootPaths.tableRoot,
            children: [
              {
                path: paths.basicTables,
                element: (
                  <Suspense key="tables" fallback={<FalconLoader />}>
                    <Tables />
                  </Suspense>
                )
              },
              {
                path: paths.advanceTables,
                element: (
                  <Suspense key="advanceTables" fallback={<FalconLoader />}>
                    <AdvanceTableExamples />
                  </Suspense>
                )
              }
            ]
          },
          {
            path: rootPaths.chartsRoot,
            children: [
              {
                path: paths.chartjs,
                element: (
                  <Suspense key="chartjs" fallback={<FalconLoader />}>
                    <Chartjs />
                  </Suspense>
                )
              },
              {
                path: paths.d3js,
                element: (
                  <Suspense key="d3j" fallback={<FalconLoader />}>
                    <D3js />
                  </Suspense>
                )
              },
              {
                path: rootPaths.echartsRoot,
                children: [
                  {
                    path: paths.echartsHowToUse,
                    element: <HowToUse />
                  },
                  {
                    path: paths.lineCharts,
                    element: (
                      <Suspense
                        key="echarts-lineChart"
                        fallback={<FalconLoader />}
                      >
                        <LineCharts />
                      </Suspense>
                    )
                  },
                  {
                    path: paths.barCharts,
                    element: (
                      <Suspense
                        key="echarts-barChart"
                        fallback={<FalconLoader />}
                      >
                        <BarCharts />
                      </Suspense>
                    )
                  },
                  {
                    path: paths.candlestickCharts,
                    element: (
                      <Suspense
                        key="echarts-candleStick"
                        fallback={<FalconLoader />}
                      >
                        <CandlestickCharts />
                      </Suspense>
                    )
                  },
                  {
                    path: paths.geoMap,
                    element: (
                      <Suspense
                        key="echarts-geoMap"
                        fallback={<FalconLoader />}
                      >
                        <GeoMaps />
                      </Suspense>
                    )
                  },
                  {
                    path: paths.scatterCharts,
                    element: (
                      <Suspense
                        key="echarts-scatterChart"
                        fallback={<FalconLoader />}
                      >
                        <ScatterCharts />
                      </Suspense>
                    )
                  },
                  {
                    path: paths.pieCharts,
                    element: (
                      <Suspense
                        key="echarts-pieChart"
                        fallback={<FalconLoader />}
                      >
                        <PieCharts />
                      </Suspense>
                    )
                  },
                  {
                    path: paths.radarCharts,
                    element: (
                      <Suspense
                        key="echarts-radarChart"
                        fallback={<FalconLoader />}
                      >
                        <RadarCharts />
                      </Suspense>
                    )
                  },
                  {
                    path: paths.heatmapCharts,
                    element: (
                      <Suspense
                        key="echarts-heatmapChart"
                        fallback={<FalconLoader />}
                      >
                        <HeatmapCharts />
                      </Suspense>
                    )
                  }
                ]
              }
            ]
          },
          {
            path: rootPaths.iconsRoot,
            children: [
              {
                path: paths.fontAwesome,
                element: <FontAwesome />
              },
              {
                path: paths.reactIcons,
                element: <ReactIcons />
              }
            ]
          },
          {
            path: rootPaths.mapsRoot,
            children: [
              {
                path: paths.googleMap,
                element: (
                  <Suspense key="googleMap" fallback={<FalconLoader />}>
                    <GoogleMapExample />
                  </Suspense>
                )
              },
              {
                path: paths.leafletMap,
                element: (
                  <Suspense key="leafletMap" fallback={<FalconLoader />}>
                    <LeafletMapExample />
                  </Suspense>
                )
              }
            ]
          },
          {
            path: rootPaths.componentsRoot,
            children: [
              {
                path: paths.alerts,
                element: (
                  <Suspense key="alerts" fallback={<FalconLoader />}>
                    <Alerts />
                  </Suspense>
                )
              },
              {
                path: paths.accordion,
                element: (
                  <Suspense key="accordion" fallback={<FalconLoader />}>
                    <Accordion />
                  </Suspense>
                )
              },
              {
                path: paths.animatedIcons,
                element: <AnimatedIcons />
              },
              {
                path: paths.background,
                element: <Backgrounds />
              },
              {
                path: paths.badges,
                element: (
                  <Suspense key="badges" fallback={<FalconLoader />}>
                    <Badges />
                  </Suspense>
                )
              },
              {
                path: paths.breadcrumbs,
                element: (
                  <Suspense key="breadcrumbs" fallback={<FalconLoader />}>
                    <Breadcrumbs />
                  </Suspense>
                )
              },
              {
                path: paths.buttons,
                element: (
                  <Suspense key="buttons" fallback={<FalconLoader />}>
                    <Buttons />
                  </Suspense>
                )
              },
              {
                path: paths.calendarExample,
                element: (
                  <Suspense key="calendarExample" fallback={<FalconLoader />}>
                    <CalendarExample />
                  </Suspense>
                )
              },
              {
                path: paths.cards,
                element: <Cards />
              },
              {
                path: paths.cards,
                element: <Cards />
              },
              {
                path: rootPaths.carouselRoot,
                children: [
                  {
                    path: paths.bootstrapCarousel,
                    element: <BootstrapCarousel />
                  },
                  {
                    path: paths.slickCarousel,
                    element: <SlickCarousel />
                  }
                ]
              },
              {
                path: paths.collapse,
                element: <Collapse />
              },
              {
                path: paths.cookieNotice,
                element: <CookieNoticeExample />
              },
              {
                path: paths.countup,
                element: <CountUp />
              },
              {
                path: paths.draggable,
                element: <DraggableExample />
              },
              {
                path: paths.dropdowns,
                element: <Dropdowns />
              },
              {
                path: paths.listGroup,
                element: <ListGroups />
              },
              {
                path: paths.modals,
                element: <Modals />
              },
              {
                path: paths.offcanvas,
                element: <Offcanvas />
              },
              {
                path: rootPaths.navsAndTabsRoot,
                children: [
                  {
                    path: paths.navs,
                    element: <Navs />
                  },
                  {
                    path: paths.navbar,
                    element: <Navbars />
                  },
                  {
                    path: paths.verticalNavbar,
                    element: <VerticalNavbar />
                  },
                  {
                    path: paths.topNavbar,
                    element: <NavBarTop />
                  },
                  {
                    path: paths.doubleTopNavbar,
                    element: <NavbarDoubleTop />
                  },
                  {
                    path: paths.comboNavbar,
                    element: <ComboNavbar />
                  },
                  {
                    path: paths.tabs,
                    element: <Tabs />
                  }
                ]
              },
              {
                path: rootPaths.picturesRoot,
                children: [
                  {
                    path: paths.avatar,
                    element: <Avatar />
                  },
                  {
                    path: paths.images,
                    element: <Image />
                  },
                  {
                    path: paths.figures,
                    element: <Figures />
                  },
                  {
                    path: paths.hoverbox,
                    element: <Hoverbox />
                  },
                  {
                    path: paths.lightbox,
                    element: <Lightbox />
                  }
                ]
              },
              {
                path: paths.progressBar,
                element: <BasicProgressBar />
              },
              {
                path: paths.pagination,
                element: <Pagination />
              },
              {
                path: paths.placeholder,
                element: <Placeholder />
              },
              {
                path: paths.popovers,
                element: <Popovers />
              },
              {
                path: paths.scrollspy,
                element: <Scrollspy />
              },
              {
                path: paths.search,
                element: <Search />
              },
              {
                path: paths.spinners,
                element: <Spinners />
              },
              {
                path: paths.timeline,
                element: <Timeline />
              },
              {
                path: paths.toasts,
                element: <Toasts />
              },
              {
                path: paths.tooltips,
                element: <Tooltips />
              },
              {
                path: paths.treeview,
                element: <TreeviewExample />
              },
              {
                path: paths.typedText,
                element: <TypedText />
              },
              {
                path: rootPaths.videosRoot,
                children: [
                  {
                    path: paths.embedVideo,
                    element: <Embed />
                  },
                  {
                    path: paths.reactPlayer,
                    element: <ReactPlayerExample />
                  }
                ]
              }
            ]
          },
          {
            path: rootPaths.utilitiesRoot,
            children: [
              {
                path: paths.backgroundColor,
                element: <Background />
              },
              {
                path: paths.borders,
                element: <Borders />
              },
              {
                path: paths.colors,
                element: <Colors />
              },
              {
                path: paths.coloredLinks,
                element: <ColoredLinks />
              },
              {
                path: paths.display,
                element: <Display />
              },
              {
                path: paths.flex,
                element: <Flex />
              },
              {
                path: paths.float,
                element: <Float />
              },
              {
                path: paths.grid,
                element: <Grid />
              },
              {
                path: paths.scrollBar,
                element: <Scrollbar />
              },
              {
                path: paths.position,
                element: <Position />
              },
              {
                path: paths.spacing,
                element: <Spacing />
              },
              {
                path: paths.sizing,
                element: <Sizing />
              },
              {
                path: paths.stretchedLink,
                element: <StretchedLink />
              },
              {
                path: paths.textTruncation,
                element: <TextTruncation />
              },
              {
                path: paths.typography,
                element: <Typography />
              },
              {
                path: paths.verticalAlign,
                element: <VerticalAlign />
              },
              {
                path: paths.visibility,
                element: <Visibility />
              }
            ]
          },
          {
            path: rootPaths.docRoot,
            children: [
              {
                path: paths.gettingStarted,
                element: <GettingStarted />
              },
              {
                path: paths.configuration,
                element: <Configuration />
              },
              {
                path: paths.styling,
                element: <Styling />
              },
              {
                path: paths.darkMode,
                element: <DarkMode />
              },
              {
                path: paths.plugin,
                element: <Plugins />
              },
              {
                path: paths.faq,
                element: <Faq />
              },
              {
                path: paths.designFile,
                element: <DesignFile />
              }
            ]
          },
          {
            path: paths.widgets,
            element: (
              <Suspense key="widgets" fallback={<FalconLoader />}>
                <Widgets />
              </Suspense>
            )
          },
          {
            path: paths.changelog,
            element: <Changelog />
          },
          {
            path: paths.migration,
            element: <Migration />
          }
        ]
      },
      {
        path: '/test',
        element: (
          <ProtectedRoute>
            <MainLayout />
          </ProtectedRoute>
        ),
        children: [
          {
            path: 'role-navigation',
            element: <RoleNavigationTester />
          }
        ]
      },
      {
        path: '/',
        element: (
          <ProtectedRoute>
            <VerticalNavLayout />
          </ProtectedRoute>
        ),
        children: [
          {
            path: paths.verticalNavLayout,
            element: <Dashboard />
          }
        ]
      },
      {
        path: '/',
        element: (
          <ProtectedRoute>
            <TopNavLayout />
          </ProtectedRoute>
        ),
        children: [
          {
            path: paths.topNavLayout,
            element: <Dashboard />
          }
        ]
      },
      {
        path: '/',
        element: (
          <ProtectedRoute>
            <ComboNavLayout />
          </ProtectedRoute>
        ),
        children: [
          {
            path: paths.comboNavLayout,
            element: <Dashboard />
          }
        ]
      },
      {
        path: '/',
        element: (
          <ProtectedRoute>
            <DoubleTopNavLayout />
          </ProtectedRoute>
        ),
        children: [
          {
            path: paths.doubleTopNavLayout,
            element: <Dashboard />
          }
        ]
      },
      {
        path: '*',
        element: <Navigate to={paths.error404} replace />
      }
    ]
  }
];

export const router = createBrowserRouter(routes, {
  basename: import.meta.env.VITE_PUBLIC_URL
});

export default routes;
