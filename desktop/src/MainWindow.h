#pragma once
#include "BackendClient.h"
#include <QJsonArray>
#include <QJsonObject>
#include <QMainWindow>
#include <QSettings>
#include <QTimer>
class QLineEdit;
class QTextEdit;
class QLabel;
class QPushButton;
class QVBoxLayout;
class QScrollArea;
class QCalendarWidget;
class QToolButton;
class QCloseEvent;
class QMessageBox;

class MainWindow final : public QMainWindow {
    Q_OBJECT
  public:
    MainWindow(const QString &backendProgram, const QString &dataDir, QWidget *parent = nullptr);

  protected:
    void closeEvent(QCloseEvent *) override;
    void resizeEvent(QResizeEvent *) override;

  private:
    BackendClient backend;
    QSettings settings;
    QWidget *navigation, *rightPanel, *composerPanel, *rows, *workspace;
    QScrollArea *scroll, *navigationScroll;
    QVBoxLayout *rowLayout;
    QLineEdit *search;
    QTextEdit *composer;
    QLabel *heading, *stagedLabel, *allCount, *favoriteCount, *sharedCount;
    QPushButton *save, *more, *leftToggle, *clearStage;
    QCalendarWidget *calendar;
    QList<QToolButton *> navButtons;
    QString page = "home", date, tag, cursor, dataPath;
    QStringList staged;
    QJsonArray tags;
    QTimer debounce;
    QPointer<QMessageBox> activeError;
    int generation = 0, searchOffset = 0, attachmentOffset = 0;
    bool compact = false, loading = false;
    void navigate(const QString &destination);
    void refresh(bool append = false);
    void metadata();
    void refreshCalendar(QCalendarWidget *widget);
    void clearRows();
    void addNote(const QJsonObject &memo);
    void editNote(const QString &uid);
    void shareNote(const QString &uid);
    void settingsPage();
    void addFiles(const QString &uid, const QStringList &paths, std::function<void()> done);
    void rpc(const QString &method, const QJsonObject &params,
             std::function<void(const QJsonValue &)> done = {});
    void applyNavigation();
    void applyAppearance();
    void updateStagedSummary();
    void showError(const QString &message);
};
