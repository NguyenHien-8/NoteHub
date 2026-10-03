#pragma once

#include <QDate>
#include <QJsonArray>
#include <QJsonObject>
#include <QMainWindow>
#include <QStringList>
#include <functional>

class IpcClient;
class QCalendarWidget;
class QCheckBox;
class QComboBox;
class QFontComboBox;
class QFrame;
class QLabel;
class QLineEdit;
class QListWidget;
class QPushButton;
class QSpinBox;
class QStackedWidget;
class QTextEdit;
class QTreeWidget;
class QVBoxLayout;
class QWidget;
class QResizeEvent;
class QCloseEvent;

class MainWindow final : public QMainWindow {
    Q_OBJECT
public:
    explicit MainWindow(IpcClient *ipc, QWidget *parent = nullptr);
    void initialize(const QJsonObject &backendInfo);
    void showForDesktop();

protected:
    void resizeEvent(QResizeEvent *event) override;
    void closeEvent(QCloseEvent *event) override;

private:
    enum class TimelineMode { All, Favorites, Shared, Tag, Date };

    void buildUi();
    QWidget *buildHomePage();
    QWidget *buildCalendarPage();
    QWidget *buildSearchPage();
    QWidget *buildAttachmentsPage();
    QWidget *buildTagsPage();
    QWidget *buildSettingsPage();
    QFrame *buildRightPanel();
    QPushButton *makeNavButton(const QString &text, const QString &compact, int pageIndex);

    void switchPage(int index);
    void toggleLeftRail();
    void updateResponsivePanels();
    void updateNavText();
    void applyUserTheme();

    void saveNewMemo();
    void chooseComposerAttachments();
    void appendTagToComposer();
    void attachFilesSequentially(qint64 memoId, const QStringList &files, std::function<void(QStringList)> done);

    void setTimelineMode(TimelineMode mode, const QString &value = {});
    void loadTimeline(bool reset = true);
    void refreshCounts();
    void refreshTags();
    void refreshCalendarMonth();
    void runSearch();
    void loadCalendarDate(const QDate &date);
    void loadAttachments();

    void addMemoCards(QVBoxLayout *layout, const QJsonArray &items);
    void clearLayout(QVBoxLayout *layout);
    void editMemo(qint64 id);
    void deleteMemo(qint64 id);
    void setFavorite(qint64 id, bool favorite);
    void shareMemo(qint64 id);
    void openAttachment(qint64 id);
    void openMemoFromAttachment(qint64 memoId);
    void showError(const QString &title, const QJsonObject &error);
    void refreshAll();

    void exportBackup();
    void importBackup();
    void updateShareServer(bool enabled);
    void refreshShareStatus();

    IpcClient *ipc_ = nullptr;
    QJsonObject backendInfo_;

    QWidget *root_ = nullptr;
    QFrame *leftPanel_ = nullptr;
    QFrame *rightPanel_ = nullptr;
    QStackedWidget *pages_ = nullptr;
    QList<QPushButton *> navButtons_;
    bool leftCollapsed_ = false;
    bool rightAutoHidden_ = false;

    QLineEdit *globalSearch_ = nullptr;
    QLabel *statusLabel_ = nullptr;

    QTextEdit *composer_ = nullptr;
    QLabel *stagedFilesLabel_ = nullptr;
    QStringList stagedFiles_;
    QVBoxLayout *timelineLayout_ = nullptr;
    QPushButton *loadMoreButton_ = nullptr;
    QJsonObject timelineCursor_;
    TimelineMode timelineMode_ = TimelineMode::All;
    QString timelineValue_;
    QLabel *homeFilterLabel_ = nullptr;

    QPushButton *allCountButton_ = nullptr;
    QPushButton *favoriteCountButton_ = nullptr;
    QPushButton *sharedCountButton_ = nullptr;
    QCalendarWidget *rightCalendar_ = nullptr;

    QCalendarWidget *calendar_ = nullptr;
    QList<QDate> calendarMarkedDates_;
    QVBoxLayout *calendarMemoLayout_ = nullptr;
    QLabel *calendarSummary_ = nullptr;

    QLineEdit *searchText_ = nullptr;
    QLineEdit *searchTags_ = nullptr;
    QLineEdit *searchFrom_ = nullptr;
    QLineEdit *searchTo_ = nullptr;
    QVBoxLayout *searchResultsLayout_ = nullptr;
    QLabel *searchSummary_ = nullptr;

    QTreeWidget *attachmentsTree_ = nullptr;
    QListWidget *tagsList_ = nullptr;

    QComboBox *appearanceCombo_ = nullptr;
    QFontComboBox *fontCombo_ = nullptr;
    QSpinBox *fontSizeSpin_ = nullptr;
    QLabel *dataDirLabel_ = nullptr;
    QCheckBox *shareEnabled_ = nullptr;
    QSpinBox *sharePort_ = nullptr;
    QLabel *shareUrlLabel_ = nullptr;
};
