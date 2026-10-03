#include "MainWindow.h"

#include "IpcClient.h"
#include "MemoCard.h"
#include "MemoDialog.h"
#include "ShareDialog.h"
#include "Theme.h"

#include <QApplication>
#include <QVariant>
#include <QKeySequence>
#include <QFont>
#include <QAbstractItemView>
#include <QButtonGroup>
#include <QCalendarWidget>
#include <QCheckBox>
#include <QClipboard>
#include <QCloseEvent>
#include <QComboBox>
#include <QDate>
#include <QDateTime>
#include <QDesktopServices>
#include <QDialog>
#include <QDialogButtonBox>
#include <QFileDialog>
#include <QFileInfo>
#include <QFontComboBox>
#include <QFrame>
#include <QGridLayout>
#include <QHBoxLayout>
#include <QTextCursor>
#include <QShortcut>
#include <QPixmap>
#include <QPalette>
#include <QJsonValue>
#include <QIcon>
#include <QInputDialog>
#include <QJsonArray>
#include <QLabel>
#include <QLineEdit>
#include <QListWidget>
#include <QMessageBox>
#include <QPushButton>
#include <QResizeEvent>
#include <QScrollArea>
#include <QSettings>
#include <QSpinBox>
#include <QStackedWidget>
#include <QStandardPaths>
#include <QTextEdit>
#include <QTextCharFormat>
#include <QTime>
#include <QTimeZone>
#include <QTreeWidget>
#include <QTreeWidgetItem>
#include <QUrl>
#include <QVBoxLayout>
#include <functional>

namespace {
qint64 jsonInt(const QJsonValue &v) { return v.toVariant().toLongLong(); }

QString errorMessage(const QJsonObject &error) {
    return error.value("message").toString(QObject::tr("Đã xảy ra lỗi không xác định."));
}

QScrollArea *scrollFor(QWidget *content, QWidget *parent = nullptr) {
    auto *scroll = new QScrollArea(parent);
    scroll->setWidgetResizable(true);
    scroll->setFrameShape(QFrame::NoFrame);
    scroll->setWidget(content);
    return scroll;
}

QWidget *scrollBody(QVBoxLayout **outLayout, QWidget *parent = nullptr) {
    auto *body = new QWidget(parent);
    auto *layout = new QVBoxLayout(body);
    layout->setContentsMargins(4, 4, 4, 4);
    layout->setSpacing(10);
    layout->setAlignment(Qt::AlignTop);
    *outLayout = layout;
    return body;
}

QString dateStartIso(const QDate &date) {
    if (!date.isValid()) return {};
    const QDateTime t(date, QTime(0, 0), QTimeZone::systemTimeZone());
    return t.toString(Qt::ISODate);
}

QString dateEndIso(const QDate &date) {
    return dateStartIso(date.addDays(1));
}

int localOffsetMinutes(const QDate &date = QDate::currentDate()) {
    // Noon avoids the rare ambiguity around a midnight DST transition. The
    // resulting offset is passed to Go so calendar boundaries match the GUI's
    // local civil day.
    const QDateTime local(date, QTime(12, 0), QTimeZone::systemTimeZone());
    return local.offsetFromUtc() / 60;
}

}

MainWindow::MainWindow(IpcClient *ipc, QWidget *parent) : QMainWindow(parent), ipc_(ipc) {
    setWindowTitle("NoteHub");
    setWindowIcon(QIcon(":/icons/NoteHub.png"));
    setMinimumSize(860, 600);
    buildUi();

    connect(ipc_, &IpcClient::backendLog, this, [this](const QString &line) {
        if (statusLabel_) statusLabel_->setToolTip(line);
    });
}

void MainWindow::initialize(const QJsonObject &backendInfo) {
    backendInfo_ = backendInfo;
    if (dataDirLabel_) {
        dataDirLabel_->setText(backendInfo_.value("dataDir").toString());
        dataDirLabel_->setToolTip(dataDirLabel_->text());
    }
    refreshAll();
    refreshShareStatus();
}

void MainWindow::showForDesktop() {
    // Apply the maximized state before the first visible native frame. Qt lets
    // the window manager fit the title bar and taskbar/work area, avoiding the
    // Fyne normal-window -> maximize flash that motivated this migration.
    setWindowState(windowState() | Qt::WindowMaximized);
    show();
}

void MainWindow::buildUi() {
    root_ = new QWidget(this);
    root_->setObjectName("root");
    setCentralWidget(root_);

    auto *rootLayout = new QVBoxLayout(root_);
    rootLayout->setContentsMargins(12, 10, 12, 10);
    rootLayout->setSpacing(10);

    auto *header = new QFrame(root_);
    header->setObjectName("panel");
    auto *headerLayout = new QHBoxLayout(header);
    headerLayout->setContentsMargins(12, 8, 12, 8);
    auto *railToggle = new QPushButton(QStringLiteral("☰"), header);
    railToggle->setFixedWidth(38);
    railToggle->setToolTip(tr("Thu gọn/mở thanh điều hướng"));
    connect(railToggle, &QPushButton::clicked, this, &MainWindow::toggleLeftRail);
    headerLayout->addWidget(railToggle);

    auto *logo = new QLabel(header);
    QPixmap icon(":/icons/NoteHub.png");
    logo->setPixmap(icon.scaled(34, 34, Qt::KeepAspectRatio, Qt::SmoothTransformation));
    headerLayout->addWidget(logo);
    auto *brand = new QLabel("NoteHub", header);
    brand->setObjectName("brand");
    headerLayout->addWidget(brand);
    headerLayout->addSpacing(16);

    globalSearch_ = new QLineEdit(header);
    globalSearch_->setPlaceholderText(tr("Tìm kiếm ghi chú…  Ctrl+K"));
    globalSearch_->setClearButtonEnabled(true);
    globalSearch_->setMaximumWidth(560);
    connect(globalSearch_, &QLineEdit::returnPressed, this, [this] {
        searchText_->setText(globalSearch_->text());
        switchPage(2);
        runSearch();
    });
    headerLayout->addWidget(globalSearch_, 1);

    statusLabel_ = new QLabel(tr("Sẵn sàng"), header);
    statusLabel_->setObjectName("muted");
    headerLayout->addWidget(statusLabel_);
    rootLayout->addWidget(header);

    auto *body = new QHBoxLayout;
    body->setSpacing(10);

    leftPanel_ = new QFrame(root_);
    leftPanel_->setObjectName("leftPanel");
    leftPanel_->setFixedWidth(220);
    auto *left = new QVBoxLayout(leftPanel_);
    left->setContentsMargins(8, 10, 8, 10);
    left->setSpacing(6);
    auto *navTitle = new QLabel(tr("ĐIỀU HƯỚNG"), leftPanel_);
    navTitle->setObjectName("muted");
    left->addWidget(navTitle);
    left->addWidget(makeNavButton(tr("Trang chủ"), "H", 0));
    left->addWidget(makeNavButton(tr("Lịch"), "C", 1));
    left->addWidget(makeNavButton(tr("Tìm kiếm"), "S", 2));
    left->addWidget(makeNavButton(tr("Tệp đính kèm"), "A", 3));
    left->addWidget(makeNavButton(tr("Thẻ"), "T", 4));
    left->addStretch();
    left->addWidget(makeNavButton(tr("Cài đặt"), "⚙", 5));
    body->addWidget(leftPanel_);

    pages_ = new QStackedWidget(root_);
    pages_->addWidget(buildHomePage());
    pages_->addWidget(buildCalendarPage());
    pages_->addWidget(buildSearchPage());
    pages_->addWidget(buildAttachmentsPage());
    pages_->addWidget(buildTagsPage());
    pages_->addWidget(buildSettingsPage());
    body->addWidget(pages_, 1);

    rightPanel_ = buildRightPanel();
    body->addWidget(rightPanel_);
    rootLayout->addLayout(body, 1);

    auto *shortcut = new QShortcut(QKeySequence::Find, this);
    connect(shortcut, &QShortcut::activated, this, [this] { globalSearch_->setFocus(); globalSearch_->selectAll(); });
    auto *ctrlK = new QShortcut(QKeySequence("Ctrl+K"), this);
    connect(ctrlK, &QShortcut::activated, this, [this] { globalSearch_->setFocus(); globalSearch_->selectAll(); });

    navButtons_.first()->setChecked(true);
    applyUserTheme();
}

QPushButton *MainWindow::makeNavButton(const QString &text, const QString &compact, int pageIndex) {
    auto *button = new QPushButton(text, leftPanel_);
    button->setCheckable(true);
    button->setProperty("fullText", text);
    button->setProperty("compactText", compact);
    button->setToolTip(text);
    button->setMinimumHeight(38);
    connect(button, &QPushButton::clicked, this, [this, pageIndex, button] {
        for (auto *other : navButtons_) other->setChecked(other == button);
        switchPage(pageIndex);
    });
    navButtons_.append(button);
    return button;
}

QWidget *MainWindow::buildHomePage() {
    auto *page = new QWidget(this);
    auto *layout = new QVBoxLayout(page);
    layout->setContentsMargins(0, 0, 0, 0);
    layout->setSpacing(10);

    auto *composerFrame = new QFrame(page);
    composerFrame->setObjectName("composer");
    auto *composerLayout = new QVBoxLayout(composerFrame);
    composerLayout->setContentsMargins(14, 12, 14, 12);
    auto *titleRow = new QHBoxLayout;
    auto *title = new QLabel(tr("Ghi chú mới"), composerFrame);
    title->setObjectName("sectionTitle");
    titleRow->addWidget(title);
    titleRow->addStretch();
    homeFilterLabel_ = new QLabel(tr("Tất cả ghi chú"), composerFrame);
    homeFilterLabel_->setObjectName("muted");
    titleRow->addWidget(homeFilterLabel_);
    composerLayout->addLayout(titleRow);

    composer_ = new QTextEdit(composerFrame);
    composer_->setAcceptRichText(false);
    composer_->setPlaceholderText(tr("Viết điều bạn muốn ghi nhớ… Hỗ trợ Markdown và #tag."));
    composer_->setMinimumHeight(96);
    composer_->setMaximumHeight(190);
    composerLayout->addWidget(composer_);

    stagedFilesLabel_ = new QLabel(composerFrame);
    stagedFilesLabel_->setObjectName("muted");
    stagedFilesLabel_->hide();
    composerLayout->addWidget(stagedFilesLabel_);

    auto *actions = new QHBoxLayout;
    auto *attach = new QPushButton(tr("Đính kèm"), composerFrame);
    auto *tag = new QPushButton(tr("Thêm tag"), composerFrame);
    auto *save = new QPushButton(tr("Lưu ghi chú"), composerFrame);
    save->setObjectName("primary");
    connect(attach, &QPushButton::clicked, this, &MainWindow::chooseComposerAttachments);
    connect(tag, &QPushButton::clicked, this, &MainWindow::appendTagToComposer);
    connect(save, &QPushButton::clicked, this, &MainWindow::saveNewMemo);
    actions->addWidget(attach);
    actions->addWidget(tag);
    actions->addStretch();
    actions->addWidget(save);
    composerLayout->addLayout(actions);
    layout->addWidget(composerFrame);

    QVBoxLayout *cards = nullptr;
    QWidget *cardsBody = scrollBody(&cards, page);
    timelineLayout_ = cards;
    layout->addWidget(scrollFor(cardsBody, page), 1);

    loadMoreButton_ = new QPushButton(tr("Tải thêm"), page);
    connect(loadMoreButton_, &QPushButton::clicked, this, [this] { loadTimeline(false); });
    loadMoreButton_->hide();
    layout->addWidget(loadMoreButton_);
    return page;
}

QWidget *MainWindow::buildCalendarPage() {
    auto *page = new QWidget(this);
    auto *layout = new QVBoxLayout(page);
    layout->setContentsMargins(0, 0, 0, 0);
    auto *panel = new QFrame(page);
    panel->setObjectName("panel");
    auto *panelLayout = new QVBoxLayout(panel);
    auto *title = new QLabel(tr("Lịch ghi chú"), panel);
    title->setObjectName("sectionTitle");
    panelLayout->addWidget(title);
    calendar_ = new QCalendarWidget(panel);
    calendar_->setGridVisible(true);
    calendar_->setVerticalHeaderFormat(QCalendarWidget::NoVerticalHeader);
    panelLayout->addWidget(calendar_);
    calendarSummary_ = new QLabel(panel);
    calendarSummary_->setObjectName("muted");
    panelLayout->addWidget(calendarSummary_);
    layout->addWidget(panel);

    QVBoxLayout *cards = nullptr;
    QWidget *body = scrollBody(&cards, page);
    calendarMemoLayout_ = cards;
    layout->addWidget(scrollFor(body, page), 1);
    connect(calendar_, &QCalendarWidget::selectionChanged, this, [this] { loadCalendarDate(calendar_->selectedDate()); });
    connect(calendar_, &QCalendarWidget::currentPageChanged, this, [this](int, int) { refreshCalendarMonth(); });
    return page;
}

QWidget *MainWindow::buildSearchPage() {
    auto *page = new QWidget(this);
    auto *layout = new QVBoxLayout(page);
    layout->setContentsMargins(0, 0, 0, 0);
    auto *panel = new QFrame(page);
    panel->setObjectName("panel");
    auto *p = new QVBoxLayout(panel);
    auto *title = new QLabel(tr("Tìm kiếm nâng cao"), panel);
    title->setObjectName("sectionTitle");
    p->addWidget(title);
    searchText_ = new QLineEdit(panel);
    searchText_->setPlaceholderText(tr("Nội dung cần tìm"));
    p->addWidget(searchText_);
    auto *filters = new QGridLayout;
    searchTags_ = new QLineEdit(panel);
    searchTags_->setPlaceholderText(tr("tag1, tag2"));
    searchFrom_ = new QLineEdit(panel);
    searchFrom_->setPlaceholderText("YYYY-MM-DD");
    searchTo_ = new QLineEdit(panel);
    searchTo_->setPlaceholderText("YYYY-MM-DD");
    filters->addWidget(new QLabel(tr("Tags"), panel), 0, 0);
    filters->addWidget(searchTags_, 0, 1);
    filters->addWidget(new QLabel(tr("Từ ngày"), panel), 0, 2);
    filters->addWidget(searchFrom_, 0, 3);
    filters->addWidget(new QLabel(tr("Đến ngày"), panel), 0, 4);
    filters->addWidget(searchTo_, 0, 5);
    p->addLayout(filters);
    auto *row = new QHBoxLayout;
    auto *search = new QPushButton(tr("Tìm"), panel);
    search->setObjectName("primary");
    searchSummary_ = new QLabel(panel);
    searchSummary_->setObjectName("muted");
    row->addWidget(search);
    row->addWidget(searchSummary_, 1);
    p->addLayout(row);
    connect(search, &QPushButton::clicked, this, &MainWindow::runSearch);
    connect(searchText_, &QLineEdit::returnPressed, this, &MainWindow::runSearch);
    layout->addWidget(panel);

    QVBoxLayout *cards = nullptr;
    QWidget *body = scrollBody(&cards, page);
    searchResultsLayout_ = cards;
    layout->addWidget(scrollFor(body, page), 1);
    return page;
}

QWidget *MainWindow::buildAttachmentsPage() {
    auto *page = new QWidget(this);
    auto *layout = new QVBoxLayout(page);
    layout->setContentsMargins(0, 0, 0, 0);
    auto *panel = new QFrame(page);
    panel->setObjectName("panel");
    auto *p = new QVBoxLayout(panel);
    auto *header = new QHBoxLayout;
    auto *title = new QLabel(tr("Tệp đính kèm"), panel);
    title->setObjectName("sectionTitle");
    header->addWidget(title);
    header->addStretch();
    auto *refresh = new QPushButton(tr("Làm mới"), panel);
    auto *open = new QPushButton(tr("Mở"), panel);
    auto *folder = new QPushButton(tr("Thư mục"), panel);
    auto *note = new QPushButton(tr("Ghi chú"), panel);
    auto *remove = new QPushButton(tr("Xóa"), panel);
    remove->setObjectName("danger");
    header->addWidget(refresh);
    header->addWidget(open);
    header->addWidget(folder);
    header->addWidget(note);
    header->addWidget(remove);
    p->addLayout(header);

    attachmentsTree_ = new QTreeWidget(panel);
    attachmentsTree_->setHeaderLabels({tr("Tên tệp"), tr("Loại"), tr("Kích thước"), tr("Ngày"), tr("Memo")});
    attachmentsTree_->setRootIsDecorated(false);
    attachmentsTree_->setAlternatingRowColors(true);
    attachmentsTree_->setSelectionMode(QAbstractItemView::SingleSelection);
    p->addWidget(attachmentsTree_, 1);
    layout->addWidget(panel, 1);

    auto current = [this]() -> QTreeWidgetItem * { return attachmentsTree_->currentItem(); };
    connect(refresh, &QPushButton::clicked, this, &MainWindow::loadAttachments);
    connect(open, &QPushButton::clicked, this, [this, current] {
        if (auto *item = current()) openAttachment(item->data(0, Qt::UserRole).toLongLong());
    });
    connect(folder, &QPushButton::clicked, this, [this, current] {
        if (auto *item = current()) {
            const QString path = item->data(0, Qt::UserRole + 2).toString();
            if (!path.isEmpty()) QDesktopServices::openUrl(QUrl::fromLocalFile(QFileInfo(path).absolutePath()));
        }
    });
    connect(note, &QPushButton::clicked, this, [this, current] {
        if (auto *item = current()) openMemoFromAttachment(item->data(0, Qt::UserRole + 1).toLongLong());
    });
    connect(remove, &QPushButton::clicked, this, [this, current] {
        auto *item = current();
        if (!item) return;
        const qint64 id = item->data(0, Qt::UserRole).toLongLong();
        if (QMessageBox::question(this, tr("Xóa tệp"), tr("Xóa vĩnh viễn tệp đính kèm này?")) != QMessageBox::Yes) return;
        ipc_->call("attachment.delete", QJsonObject{{"id", id}}, [this](const QJsonValue &, const QJsonObject &error) {
            if (!error.isEmpty()) showError(tr("Không thể xóa tệp"), error); else refreshAll();
        });
    });
    connect(attachmentsTree_, &QTreeWidget::itemDoubleClicked, this, [this](QTreeWidgetItem *item, int) {
        openAttachment(item->data(0, Qt::UserRole).toLongLong());
    });
    return page;
}

QWidget *MainWindow::buildTagsPage() {
    auto *page = new QWidget(this);
    auto *layout = new QVBoxLayout(page);
    layout->setContentsMargins(0, 0, 0, 0);
    auto *panel = new QFrame(page);
    panel->setObjectName("panel");
    auto *p = new QVBoxLayout(panel);
    auto *title = new QLabel(tr("Tags"), panel);
    title->setObjectName("sectionTitle");
    p->addWidget(title);
    auto *hint = new QLabel(tr("Thêm #tag vào nội dung. Dùng / để tạo nhóm như #Research/FPGA."), panel);
    hint->setObjectName("muted");
    p->addWidget(hint);
    tagsList_ = new QListWidget(panel);
    p->addWidget(tagsList_, 1);
    layout->addWidget(panel, 1);
    connect(tagsList_, &QListWidget::itemActivated, this, [this](QListWidgetItem *item) {
        setTimelineMode(TimelineMode::Tag, item->data(Qt::UserRole).toString());
        switchPage(0);
    });
    return page;
}

QWidget *MainWindow::buildSettingsPage() {
    auto *page = new QWidget(this);
    auto *layout = new QVBoxLayout(page);
    layout->setContentsMargins(0, 0, 0, 0);
    auto *panel = new QFrame(page);
    panel->setObjectName("panel");
    auto *p = new QVBoxLayout(panel);
    auto *title = new QLabel(tr("Cài đặt"), panel);
    title->setObjectName("sectionTitle");
    p->addWidget(title);

    auto *grid = new QGridLayout;
    grid->setColumnStretch(1, 1);
    appearanceCombo_ = new QComboBox(panel);
    appearanceCombo_->addItem(tr("Theo hệ thống"), "system");
    appearanceCombo_->addItem(tr("Sáng"), "light");
    appearanceCombo_->addItem(tr("Tối"), "dark");
    fontCombo_ = new QFontComboBox(panel);
    fontSizeSpin_ = new QSpinBox(panel);
    fontSizeSpin_->setRange(8, 24);
    fontSizeSpin_->setSuffix(" pt");
    grid->addWidget(new QLabel(tr("Giao diện"), panel), 0, 0);
    grid->addWidget(appearanceCombo_, 0, 1);
    grid->addWidget(new QLabel(tr("Font"), panel), 1, 0);
    grid->addWidget(fontCombo_, 1, 1);
    grid->addWidget(new QLabel(tr("Cỡ chữ"), panel), 2, 0);
    grid->addWidget(fontSizeSpin_, 2, 1);

    dataDirLabel_ = new QLabel(panel);
    dataDirLabel_->setTextInteractionFlags(Qt::TextSelectableByMouse);
    dataDirLabel_->setWordWrap(true);
    auto *openData = new QPushButton(tr("Mở thư mục dữ liệu"), panel);
    grid->addWidget(new QLabel(tr("Dữ liệu"), panel), 3, 0);
    grid->addWidget(dataDirLabel_, 3, 1);
    grid->addWidget(openData, 3, 2);
    p->addLayout(grid);

    auto *backupRow = new QHBoxLayout;
    auto *exportButton = new QPushButton(tr("Xuất backup ZIP"), panel);
    auto *importButton = new QPushButton(tr("Nhập backup ZIP"), panel);
    backupRow->addWidget(exportButton);
    backupRow->addWidget(importButton);
    backupRow->addStretch();
    p->addLayout(backupRow);

    auto *shareTitle = new QLabel(tr("Chia sẻ cục bộ"), panel);
    shareTitle->setObjectName("sectionTitle");
    p->addWidget(shareTitle);
    auto *shareRow = new QHBoxLayout;
    shareEnabled_ = new QCheckBox(tr("Bật server 127.0.0.1"), panel);
    sharePort_ = new QSpinBox(panel);
    sharePort_->setRange(1024, 65535);
    sharePort_->setValue(8787);
    shareUrlLabel_ = new QLabel(panel);
    shareUrlLabel_->setObjectName("muted");
    shareUrlLabel_->setTextInteractionFlags(Qt::TextSelectableByMouse);
    shareRow->addWidget(shareEnabled_);
    shareRow->addWidget(new QLabel(tr("Port"), panel));
    shareRow->addWidget(sharePort_);
    shareRow->addWidget(shareUrlLabel_, 1);
    p->addLayout(shareRow);

    auto *version = new QLabel(tr("Qt GUI + Go Backend · IPC JSON/stdio · NoteHub 0.3.0"), panel);
    version->setObjectName("muted");
    p->addWidget(version);
    p->addStretch();
    layout->addWidget(panel, 1);

    QSettings settings("NoteHub", "NoteHub");
    const QString mode = settings.value("appearance", "system").toString();
    int appearanceIndex = appearanceCombo_->findData(mode);
    if (appearanceIndex >= 0) appearanceCombo_->setCurrentIndex(appearanceIndex);
    const QString family = settings.value("fontFamily", QApplication::font().family()).toString();
    fontCombo_->setCurrentFont(QFont(family));
    fontSizeSpin_->setValue(settings.value("fontSize", QApplication::font().pointSize()).toInt());
    sharePort_->setValue(settings.value("sharePort", 8787).toInt());

    connect(appearanceCombo_, &QComboBox::currentIndexChanged, this, [this] { applyUserTheme(); });
    connect(fontCombo_, &QFontComboBox::currentFontChanged, this, [this](const QFont &) { applyUserTheme(); });
    connect(fontSizeSpin_, qOverload<int>(&QSpinBox::valueChanged), this, [this](int) { applyUserTheme(); });
    connect(openData, &QPushButton::clicked, this, [this] {
        const QString path = backendInfo_.value("dataDir").toString();
        if (!path.isEmpty()) QDesktopServices::openUrl(QUrl::fromLocalFile(path));
    });
    connect(exportButton, &QPushButton::clicked, this, &MainWindow::exportBackup);
    connect(importButton, &QPushButton::clicked, this, &MainWindow::importBackup);
    connect(shareEnabled_, &QCheckBox::toggled, this, &MainWindow::updateShareServer);
    connect(sharePort_, qOverload<int>(&QSpinBox::valueChanged), this, [this](int port) {
        QSettings("NoteHub", "NoteHub").setValue("sharePort", port);
        if (shareEnabled_->isChecked()) updateShareServer(true);
    });
    return page;
}

QFrame *MainWindow::buildRightPanel() {
    auto *panel = new QFrame(this);
    panel->setObjectName("rightPanel");
    panel->setFixedWidth(280);
    auto *layout = new QVBoxLayout(panel);
    layout->setContentsMargins(10, 10, 10, 10);
    auto *title = new QLabel(tr("Bộ lọc nhanh"), panel);
    title->setObjectName("sectionTitle");
    layout->addWidget(title);
    allCountButton_ = new QPushButton(tr("Tất cả"), panel);
    favoriteCountButton_ = new QPushButton(tr("Yêu thích"), panel);
    sharedCountButton_ = new QPushButton(tr("Đã chia sẻ"), panel);
    connect(allCountButton_, &QPushButton::clicked, this, [this] { setTimelineMode(TimelineMode::All); switchPage(0); });
    connect(favoriteCountButton_, &QPushButton::clicked, this, [this] { setTimelineMode(TimelineMode::Favorites); switchPage(0); });
    connect(sharedCountButton_, &QPushButton::clicked, this, [this] { setTimelineMode(TimelineMode::Shared); switchPage(0); });
    layout->addWidget(allCountButton_);
    layout->addWidget(favoriteCountButton_);
    layout->addWidget(sharedCountButton_);
    layout->addSpacing(8);
    auto *calTitle = new QLabel(tr("Lịch"), panel);
    calTitle->setObjectName("sectionTitle");
    layout->addWidget(calTitle);
    rightCalendar_ = new QCalendarWidget(panel);
    rightCalendar_->setGridVisible(false);
    rightCalendar_->setVerticalHeaderFormat(QCalendarWidget::NoVerticalHeader);
    rightCalendar_->setMaximumHeight(250);
    layout->addWidget(rightCalendar_);
    connect(rightCalendar_, &QCalendarWidget::selectionChanged, this, [this] {
        setTimelineMode(TimelineMode::Date, rightCalendar_->selectedDate().toString(Qt::ISODate));
        switchPage(0);
    });
    layout->addStretch();
    return panel;
}

void MainWindow::switchPage(int index) {
    if (index < 0 || index >= pages_->count()) return;
    pages_->setCurrentIndex(index);
    for (int i = 0; i < navButtons_.size(); ++i) navButtons_[i]->setChecked(i == index);
    if (index == 1) { refreshCalendarMonth(); loadCalendarDate(calendar_->selectedDate()); }
    else if (index == 3) loadAttachments();
    else if (index == 4) refreshTags();
}

void MainWindow::toggleLeftRail() {
    leftCollapsed_ = !leftCollapsed_;
    updateNavText();
    updateResponsivePanels();
}

void MainWindow::updateNavText() {
    for (auto *button : navButtons_) {
        button->setText(leftCollapsed_ ? button->property("compactText").toString() : button->property("fullText").toString());
    }
}

void MainWindow::updateResponsivePanels() {
    const bool narrow = width() < 1120;
    rightAutoHidden_ = narrow;
    rightPanel_->setVisible(!rightAutoHidden_);
    const bool forceCompact = width() < 940;
    leftPanel_->setFixedWidth((leftCollapsed_ || forceCompact) ? 66 : 220);
    if (forceCompact && !leftCollapsed_) {
        for (auto *button : navButtons_) button->setText(button->property("compactText").toString());
    } else updateNavText();
}

void MainWindow::resizeEvent(QResizeEvent *event) {
    QMainWindow::resizeEvent(event);
    updateResponsivePanels();
}

void MainWindow::closeEvent(QCloseEvent *event) {
    QSettings settings("NoteHub", "NoteHub");
    settings.setValue("geometry", saveGeometry());
    settings.setValue("windowState", saveState());
    ipc_->stop();
    QMainWindow::closeEvent(event);
}

void MainWindow::applyUserTheme() {
    if (!appearanceCombo_ || !fontCombo_ || !fontSizeSpin_) return;
    QSettings settings("NoteHub", "NoteHub");
    const QString mode = appearanceCombo_->currentData().toString();
    const QString family = fontCombo_->currentFont().family();
    const int size = fontSizeSpin_->value();
    settings.setValue("appearance", mode);
    settings.setValue("fontFamily", family);
    settings.setValue("fontSize", size);
    Theme::apply(*qApp, mode, family, size);
}

void MainWindow::chooseComposerAttachments() {
    const QStringList files = QFileDialog::getOpenFileNames(this, tr("Chọn tệp đính kèm"));
    if (files.isEmpty()) return;
    stagedFiles_ = files;
    QStringList names;
    for (const QString &path : files) names << QFileInfo(path).fileName();
    stagedFilesLabel_->setText(tr("Sẽ đính kèm: %1").arg(names.join(", ")));
    stagedFilesLabel_->show();
}

void MainWindow::appendTagToComposer() {
    bool ok = false;
    QString tag = QInputDialog::getText(this, tr("Thêm tag"), tr("Tag (không cần #):"), QLineEdit::Normal, {}, &ok).trimmed();
    if (!ok || tag.isEmpty()) return;
    tag.remove(' ');
    QString text = composer_->toPlainText();
    if (!text.isEmpty() && !text.endsWith(' ')) text += ' ';
    text += '#' + tag;
    composer_->setPlainText(text);
    composer_->moveCursor(QTextCursor::End);
}

void MainWindow::saveNewMemo() {
    const QString content = composer_->toPlainText();
    if (content.trimmed().isEmpty() && stagedFiles_.isEmpty()) {
        QMessageBox::information(this, tr("Ghi chú trống"), tr("Hãy nhập nội dung hoặc chọn tệp đính kèm."));
        return;
    }
    statusLabel_->setText(tr("Đang lưu…"));
    ipc_->call("memo.create", QJsonObject{{"content", content}}, [this](const QJsonValue &result, const QJsonObject &error) {
        if (!error.isEmpty()) { showError(tr("Không thể lưu ghi chú"), error); statusLabel_->setText(tr("Lỗi")); return; }
        const qint64 memoId = jsonInt(result.toObject().value("id"));
        const QStringList files = stagedFiles_;
        attachFilesSequentially(memoId, files, [this](QStringList failures) {
            composer_->clear();
            stagedFiles_.clear();
            stagedFilesLabel_->hide();
            statusLabel_->setText(failures.isEmpty() ? tr("Đã lưu") : tr("Đã lưu, có tệp lỗi"));
            if (!failures.isEmpty()) QMessageBox::warning(this, tr("Một số tệp không thể đính kèm"), failures.join("\n"));
            refreshAll();
        });
    });
}

void MainWindow::attachFilesSequentially(qint64 memoId, const QStringList &files, std::function<void(QStringList)> done) {
    if (files.isEmpty()) { done({}); return; }
    auto *index = new int(0);
    auto *failures = new QStringList;
    auto *step = new std::function<void()>;
    *step = [this, memoId, files, done, index, failures, step]() {
        if (*index >= files.size()) {
            const QStringList out = *failures;
            delete index; delete failures; delete step;
            done(out);
            return;
        }
        const QString path = files.at((*index)++);
        ipc_->call("attachment.add", QJsonObject{{"memoId", memoId}, {"path", path}}, [path, failures, step](const QJsonValue &, const QJsonObject &error) {
            if (!error.isEmpty()) failures->append(QFileInfo(path).fileName() + ": " + errorMessage(error));
            (*step)();
        });
    };
    (*step)();
}

void MainWindow::setTimelineMode(TimelineMode mode, const QString &value) {
    timelineMode_ = mode;
    timelineValue_ = value;
    switch (mode) {
    case TimelineMode::All: homeFilterLabel_->setText(tr("Tất cả ghi chú")); break;
    case TimelineMode::Favorites: homeFilterLabel_->setText(tr("Yêu thích")); break;
    case TimelineMode::Shared: homeFilterLabel_->setText(tr("Đã chia sẻ")); break;
    case TimelineMode::Tag: homeFilterLabel_->setText("#" + value); break;
    case TimelineMode::Date: homeFilterLabel_->setText(tr("Ngày %1").arg(value)); break;
    }
    loadTimeline(true);
}

void MainWindow::loadTimeline(bool reset) {
    if (reset) {
        timelineCursor_ = {};
        clearLayout(timelineLayout_);
    }
    QJsonObject params{{"limit", 20}};
    if (!timelineCursor_.isEmpty()) params.insert("cursor", timelineCursor_);
    if (timelineMode_ == TimelineMode::Favorites) params.insert("favoriteOnly", true);
    if (timelineMode_ == TimelineMode::Shared) params.insert("sharedOnly", true);
    if (timelineMode_ == TimelineMode::Tag) params.insert("tags", QJsonArray{timelineValue_});
    if (timelineMode_ == TimelineMode::Date) {
        const QDate date = QDate::fromString(timelineValue_, Qt::ISODate);
        params.insert("from", dateStartIso(date));
        params.insert("to", dateEndIso(date));
    }
    loadMoreButton_->setEnabled(false);
    statusLabel_->setText(tr("Đang tải…"));
    ipc_->call("timeline.list", params, [this](const QJsonValue &result, const QJsonObject &error) {
        loadMoreButton_->setEnabled(true);
        if (!error.isEmpty()) { showError(tr("Không thể tải timeline"), error); return; }
        const QJsonObject object = result.toObject();
        addMemoCards(timelineLayout_, object.value("items").toArray());
        timelineCursor_ = object.value("next").toObject();
        loadMoreButton_->setVisible(!timelineCursor_.isEmpty());
        statusLabel_->setText(tr("Sẵn sàng"));
    });
}

void MainWindow::refreshCounts() {
    ipc_->call("timeline.counts", {}, [this](const QJsonValue &result, const QJsonObject &error) {
        if (!error.isEmpty()) return;
        const QJsonObject c = result.toObject();
        allCountButton_->setText(tr("Tất cả  ·  %1").arg(c.value("all").toInt()));
        favoriteCountButton_->setText(tr("Yêu thích  ·  %1").arg(c.value("favorites").toInt()));
        sharedCountButton_->setText(tr("Đã chia sẻ  ·  %1").arg(c.value("shared").toInt()));
    });
}

void MainWindow::refreshTags() {
    ipc_->call("tags.list", {}, [this](const QJsonValue &result, const QJsonObject &error) {
        if (!error.isEmpty()) return;
        tagsList_->clear();
        for (const QJsonValue &value : result.toObject().value("items").toArray()) {
            const QJsonObject tag = value.toObject();
            auto *item = new QListWidgetItem(QString("#%1    %2").arg(tag.value("tag").toString()).arg(tag.value("count").toInt()), tagsList_);
            item->setData(Qt::UserRole, tag.value("tag").toString());
        }
    });
}

void MainWindow::refreshCalendarMonth() {
    if (!calendar_) return;
    const int year = calendar_->yearShown();
    const int month = calendar_->monthShown();
    ipc_->call("calendar.month", QJsonObject{{"year", year}, {"month", month}, {"offsetMinutes", localOffsetMinutes(QDate(year, month, 1))}},
               [this](const QJsonValue &result, const QJsonObject &error) {
        if (!error.isEmpty()) return;
        for (const QDate &date : calendarMarkedDates_) calendar_->setDateTextFormat(date, QTextCharFormat{});
        calendarMarkedDates_.clear();

        int total = 0;
        QTextCharFormat marked;
        marked.setFontWeight(QFont::Bold);
        marked.setForeground(qApp->palette().color(QPalette::Highlight));
        for (const QJsonValue &v : result.toObject().value("days").toArray()) {
            const QJsonObject day = v.toObject();
            total += day.value("count").toInt();
            const QDate date = QDate::fromString(day.value("date").toString(), Qt::ISODate);
            if (date.isValid()) {
                calendar_->setDateTextFormat(date, marked);
                calendarMarkedDates_.append(date);
            }
        }
        calendarSummary_->setText(tr("%1 ghi chú trong tháng này").arg(total));
    });
}

void MainWindow::runSearch() {
    QJsonObject params{{"text", searchText_->text()}, {"limit", 100}, {"offset", 0}};
    const QStringList tagList = searchTags_->text().split(',', Qt::SkipEmptyParts);
    QJsonArray tags;
    for (QString tag : tagList) {
        tag = tag.trimmed(); if (tag.startsWith('#')) tag.remove(0, 1); if (!tag.isEmpty()) tags.append(tag);
    }
    if (!tags.isEmpty()) params.insert("tags", tags);
    const QDate from = QDate::fromString(searchFrom_->text().trimmed(), Qt::ISODate);
    const QDate to = QDate::fromString(searchTo_->text().trimmed(), Qt::ISODate);
    if (!searchFrom_->text().trimmed().isEmpty()) {
        if (!from.isValid()) { QMessageBox::warning(this, tr("Ngày không hợp lệ"), tr("Từ ngày phải có dạng YYYY-MM-DD.")); return; }
        params.insert("from", dateStartIso(from));
    }
    if (!searchTo_->text().trimmed().isEmpty()) {
        if (!to.isValid()) { QMessageBox::warning(this, tr("Ngày không hợp lệ"), tr("Đến ngày phải có dạng YYYY-MM-DD.")); return; }
        params.insert("to", dateEndIso(to));
    }
    if (from.isValid() && to.isValid() && from > to) {
        QMessageBox::warning(this, tr("Khoảng ngày không hợp lệ"), tr("Từ ngày không được sau Đến ngày."));
        return;
    }
    clearLayout(searchResultsLayout_);
    searchSummary_->setText(tr("Đang tìm…"));
    ipc_->call("search.query", params, [this](const QJsonValue &result, const QJsonObject &error) {
        if (!error.isEmpty()) { showError(tr("Tìm kiếm thất bại"), error); return; }
        const QJsonArray items = result.toObject().value("items").toArray();
        addMemoCards(searchResultsLayout_, items);
        searchSummary_->setText(tr("%1 kết quả").arg(items.size()));
    });
}

void MainWindow::loadCalendarDate(const QDate &date) {
    if (!date.isValid()) return;
    clearLayout(calendarMemoLayout_);
    ipc_->call("calendar.date", QJsonObject{{"date", date.toString(Qt::ISODate)}, {"limit", 100}, {"offsetMinutes", localOffsetMinutes(date)}},
               [this, date](const QJsonValue &result, const QJsonObject &error) {
        if (!error.isEmpty()) { showError(tr("Không thể tải ngày"), error); return; }
        const QJsonArray items = result.toObject().value("items").toArray();
        addMemoCards(calendarMemoLayout_, items);
        calendarSummary_->setText(tr("%1 · %2 ghi chú").arg(date.toString("dd/MM/yyyy")).arg(items.size()));
    });
}

void MainWindow::loadAttachments() {
    attachmentsTree_->clear();
    ipc_->call("attachment.listAll", QJsonObject{{"limit", 500}, {"offset", 0}}, [this](const QJsonValue &result, const QJsonObject &error) {
        if (!error.isEmpty()) { showError(tr("Không thể tải tệp"), error); return; }
        for (const QJsonValue &value : result.toObject().value("items").toArray()) {
            const QJsonObject a = value.toObject();
            const qint64 size = jsonInt(a.value("size"));
            QString sizeText = size < 1024 * 1024 ? QString::number(size / 1024.0, 'f', 1) + " KB" : QString::number(size / 1048576.0, 'f', 1) + " MB";
            const QDateTime created = QDateTime::fromString(a.value("createdAt").toString(), Qt::ISODate).toLocalTime();
            auto *item = new QTreeWidgetItem(attachmentsTree_, {a.value("filename").toString(), a.value("mimeType").toString(), sizeText, created.toString("dd/MM/yyyy HH:mm"), QString::number(jsonInt(a.value("memoId")))});
            item->setData(0, Qt::UserRole, QVariant::fromValue<qlonglong>(jsonInt(a.value("id"))));
            item->setData(0, Qt::UserRole + 1, QVariant::fromValue<qlonglong>(jsonInt(a.value("memoId"))));
            item->setData(0, Qt::UserRole + 2, a.value("localPath").toString());
        }
        for (int i = 0; i < attachmentsTree_->columnCount(); ++i) attachmentsTree_->resizeColumnToContents(i);
    });
}

void MainWindow::addMemoCards(QVBoxLayout *layout, const QJsonArray &items) {
    for (const QJsonValue &value : items) {
        auto *card = new MemoCard(value.toObject());
        connect(card, &MemoCard::editRequested, this, &MainWindow::editMemo);
        connect(card, &MemoCard::deleteRequested, this, &MainWindow::deleteMemo);
        connect(card, &MemoCard::favoriteRequested, this, &MainWindow::setFavorite);
        connect(card, &MemoCard::shareRequested, this, &MainWindow::shareMemo);
        connect(card, &MemoCard::attachmentOpenRequested, this, &MainWindow::openAttachment);
        layout->addWidget(card);
    }
    if (items.isEmpty() && layout->count() == 0) {
        auto *empty = new QLabel(tr("Không có ghi chú phù hợp."));
        empty->setObjectName("muted");
        empty->setAlignment(Qt::AlignCenter);
        empty->setMinimumHeight(80);
        layout->addWidget(empty);
    }
}

void MainWindow::clearLayout(QVBoxLayout *layout) {
    while (QLayoutItem *item = layout->takeAt(0)) {
        if (QWidget *widget = item->widget()) widget->deleteLater();
        delete item;
    }
}

void MainWindow::editMemo(qint64 id) {
    ipc_->call("memo.get", QJsonObject{{"id", id}}, [this](const QJsonValue &result, const QJsonObject &error) {
        if (!error.isEmpty()) { showError(tr("Không thể mở ghi chú"), error); return; }
        auto *dialog = new MemoDialog(ipc_, result.toObject(), this);
        connect(dialog, &MemoDialog::changed, this, &MainWindow::refreshAll);
        dialog->setAttribute(Qt::WA_DeleteOnClose);
        dialog->show();
    });
}

void MainWindow::deleteMemo(qint64 id) {
    if (QMessageBox::question(this, tr("Xóa ghi chú"), tr("Xóa ghi chú và toàn bộ tệp đính kèm? Hành động này không thể hoàn tác.")) != QMessageBox::Yes) return;
    ipc_->call("memo.delete", QJsonObject{{"id", id}}, [this](const QJsonValue &, const QJsonObject &error) {
        if (!error.isEmpty()) showError(tr("Không thể xóa ghi chú"), error); else refreshAll();
    });
}

void MainWindow::setFavorite(qint64 id, bool favorite) {
    ipc_->call("memo.favorite", QJsonObject{{"id", id}, {"favorite", favorite}}, [this](const QJsonValue &, const QJsonObject &error) {
        if (!error.isEmpty()) showError(tr("Không thể cập nhật yêu thích"), error); else refreshAll();
    });
}

void MainWindow::shareMemo(qint64 id) {
    auto *dialog = new ShareDialog(ipc_, id, this);
    dialog->setAttribute(Qt::WA_DeleteOnClose);
    connect(dialog, &ShareDialog::changed, this, &MainWindow::refreshAll);
    dialog->show();
}

void MainWindow::openAttachment(qint64 id) {
    ipc_->call("attachment.path", QJsonObject{{"id", id}}, [this](const QJsonValue &result, const QJsonObject &error) {
        if (!error.isEmpty()) { showError(tr("Không thể mở tệp"), error); return; }
        const QString path = result.toObject().value("path").toString();
        if (!QDesktopServices::openUrl(QUrl::fromLocalFile(path))) QMessageBox::warning(this, tr("Không thể mở tệp"), path);
    });
}

void MainWindow::openMemoFromAttachment(qint64 memoId) { editMemo(memoId); }

void MainWindow::showError(const QString &title, const QJsonObject &error) {
    QMessageBox::critical(this, title, errorMessage(error));
    statusLabel_->setText(tr("Lỗi"));
}

void MainWindow::refreshAll() {
    loadTimeline(true);
    refreshCounts();
    refreshTags();
    refreshCalendarMonth();
    if (pages_->currentIndex() == 1) loadCalendarDate(calendar_->selectedDate());
    if (pages_->currentIndex() == 3) loadAttachments();
}

void MainWindow::exportBackup() {
    const QString suggested = QStandardPaths::writableLocation(QStandardPaths::DocumentsLocation) + "/NoteHub-backup-" + QDate::currentDate().toString("yyyyMMdd") + ".zip";
    const QString path = QFileDialog::getSaveFileName(this, tr("Xuất backup"), suggested, tr("ZIP (*.zip)"));
    if (path.isEmpty()) return;
    if (QFileInfo::exists(path)) {
        QMessageBox::warning(this, tr("Tệp đã tồn tại"), tr("Vì an toàn, NoteHub không ghi đè backup. Hãy chọn tên tệp mới."));
        return;
    }
    ipc_->call("backup.export", QJsonObject{{"path", path}}, [this, path](const QJsonValue &, const QJsonObject &error) {
        if (!error.isEmpty()) showError(tr("Xuất backup thất bại"), error);
        else QMessageBox::information(this, tr("Đã xuất backup"), path);
    });
}

void MainWindow::importBackup() {
    const QString path = QFileDialog::getOpenFileName(this, tr("Nhập backup"), {}, tr("ZIP (*.zip)"));
    if (path.isEmpty()) return;
    const QStringList labels{tr("Bỏ qua bản trùng"), tr("Thay thế bản trùng"), tr("Tạo bản sao")};
    bool ok = false;
    const QString chosen = QInputDialog::getItem(this, tr("Xử lý xung đột"), tr("Khi UID ghi chú đã tồn tại:"), labels, 0, false, &ok);
    if (!ok) return;
    QString policy = "skip";
    if (chosen == labels[1]) policy = "replace"; else if (chosen == labels[2]) policy = "duplicate";
    ipc_->call("backup.import", QJsonObject{{"path", path}, {"policy", policy}}, [this](const QJsonValue &result, const QJsonObject &error) {
        if (!error.isEmpty()) { showError(tr("Nhập backup thất bại"), error); return; }
        const QJsonObject r = result.toObject();
        QMessageBox::information(this, tr("Đã nhập backup"), tr("Tạo mới: %1\nThay thế: %2\nNhân bản: %3\nBỏ qua: %4\nLỗi: %5")
            .arg(r.value("created").toInt()).arg(r.value("replaced").toInt()).arg(r.value("duplicated").toInt()).arg(r.value("skipped").toInt()).arg(r.value("failures").toArray().size()));
        refreshAll();
    });
}

void MainWindow::updateShareServer(bool enabled) {
    if (!ipc_->isRunning()) return;
    QSettings("NoteHub", "NoteHub").setValue("sharePort", sharePort_->value());
    ipc_->call("share.server", QJsonObject{{"enabled", enabled}, {"port", sharePort_->value()}}, [this, enabled](const QJsonValue &result, const QJsonObject &error) {
        if (!error.isEmpty()) {
            shareEnabled_->blockSignals(true); shareEnabled_->setChecked(!enabled); shareEnabled_->blockSignals(false);
            showError(tr("Không thể thay đổi server chia sẻ"), error);
            return;
        }
        const QJsonObject status = result.toObject();
        shareUrlLabel_->setText(status.value("url").toString());
    });
}

void MainWindow::refreshShareStatus() {
    if (!ipc_->isRunning() || !shareEnabled_) return;
    ipc_->call("share.status", {}, [this](const QJsonValue &result, const QJsonObject &error) {
        if (!error.isEmpty()) return;
        const QJsonObject status = result.toObject();
        shareEnabled_->blockSignals(true);
        shareEnabled_->setChecked(status.value("enabled").toBool());
        shareEnabled_->blockSignals(false);
        shareUrlLabel_->setText(status.value("url").toString());
    });
}
