/*
Copyright (C) 2023-2026 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as published
by the Free Software Foundation, either version 3 of the License, or
(at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
GNU Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License
along with this program. If not, see <https://www.gnu.org/licenses/>.

For commercial licensing, please contact support@quantumnous.com
*/
import { ArrowRight, Sparkles } from 'lucide-react'
import { useTranslation } from 'react-i18next'

import { AnimateInView } from '@/components/animate-in-view'
import { Button } from '@/components/ui/button'

type CreationCard = {
  title: string
  prompt: string
  image: string
  tags: string[]
  tone: 'amber' | 'coral' | 'sky' | 'plum'
}

// Original content from the prompt source used by the integrated canvas.
// It is kept locally so the homepage does not depend on the remote registry.
const canvasPromptImageBase =
  'https://cdn.jsdelivr.net/gh/glidea/banana-prompt-quicker@main/images'

const creationCards: CreationCard[] = [
  {
    title: '苹果风格海报',
    prompt:
      '充分参考图片的设计风格，配色等，为如下内容生成苹果风格的海报：\n\nBanana Prompt Quicker v1.6.0 1月6号震撼来袭\n全新参考图功能，去他丫的‘反推’',
    image: `${canvasPromptImageBase}/apple.png`,
    tags: ['工作', '海报', 'Official'],
    tone: 'amber',
  },
  {
    title: '疯狂动物城海报',
    prompt:
      '加载并使用 Nano Banana Pro 工具作画，而不是分析或给提示词\n---\n\n充分参考图片画风和人物形象，为如下内容画一幅宣传海报图片（3：2 竖屏风格）\n\nBanana Prompt Quicker v1.6 更新：提示词支持添加参考图\n看到心动的 PPT 风格 → 上传参考图即可复刻\n发现惊艳的滤镜效果 → 一键给照片加同款\n从此告别绞尽脑汁写 Prompt 的日子\n🔗 Chrome Web Store 搜索安装',
    image: `${canvasPromptImageBase}/dongwucheng.jpg`,
    tags: ['工作', '海报', 'Official'],
    tone: 'coral',
  },
  {
    title: '贴吧老哥疯狂吐槽批注',
    prompt:
      '生成图片，把它打印出来，然后用红墨水疯狂地加上手写中文批注、涂鸦、乱画，如果你想的话，检索这个账户内容，涂鸦的内容主要为吐槽他，用贴吧老哥的口语疯狂吐槽。还可以加点小剪贴画。',
    image: `${canvasPromptImageBase}/reddit_style_handwrite_annotation.jpg`,
    tags: ['有趣', '吐槽', '@canghecode'],
    tone: 'sky',
  },
  {
    title: '锐评世间万物',
    prompt:
      '你是一个拥有实时网络搜索能力和顶尖数据可视化设计能力的AI专家。请执行以下两个步骤：\n调研阶段：立刻针对用户指定的【2025 中国新能源汽车】进行全面的网络调研。搜集关于该领域内不同子产品、型号或作品的大众口碑、市场热度、专业评测及用户反馈数据。\n可视化阶段：基于你的调研结果，设计一张专业的信息图表（Infographic）。你需要将调研到的具体项目，精准地分类填入下面定义的五个“从夯到拉”的视觉等级模块中。\n\n【用户指定目标领域/产品】\n[在此处填写你需要调研的内容，例如：2024年热门智能手机、市面上的无糖茶饮料品牌、近十年的漫威电影、程序员常用的代码编辑器]\n\n【图像设计要求】\n整体风格：\n一张结构清晰、现代感强的模块化信息图表，采用“Bento Grid”（便当盒网格）布局。背景干净简洁，聚焦于内容呈现。视觉上必须体现出从高到低的强烈层级落差感。\n等级结构与视觉定义（严格执行以下五级）：\n\n第1级（最高层）：夯 (Hāng)\n调研填充标准：根据调研，该领域内目前公认的“版本之子”、具有统治级热度、无可争议的顶流产品/作品。\n视觉表现：占据画面最上方或最大的版面模块。色调为极具爆发力的爆裂红与辉煌金，带有光晕或能量外溢的视觉特效。字体最大、最粗。模块内需展示调研到的代表性产品的名称或高质量图像，并配以极简的赞美短语（如“全网吹爆”、“神作”）。\n\n第2级：顶级\n调研填充标准：硬核实力派，虽然热度可能不及“夯”，但口碑极佳，是行家首选的优质项目。\n视觉表现：位于第二层。色调为坚实、高级的燃烧橙与金属银。模块设计显得扎实、富有质感。展示代表性实力派产品。\n\n第3级：人上人\n调研填充标准：优越之选，品味在线，买了/看了绝对不亏的中坚力量，代表了一定的鉴赏力。\n视觉表现：位于中层。色调为明亮、干净的柠檬黄与冷灰。设计风格现代、清爽。展示代表性优质中产产品。\n\n第4级：NPC\n调研填充标准：毫无记忆点的大众脸产品，凑数的工业流水线产物，无功无过，容易被遗忘，必须要写上具体的产品或品牌或者人名不要含糊其辞。\n视觉表现：位于中下层。色调为平淡乏味的面包色/米色或纸板棕。模块设计显得普通、重复、缺乏个性。展示那些非常平庸的产品。\n\n第5级（最底层）：拉完了\n调研填充标准：调研中发现的公认“避雷针”、“智商税”、灾难级失败产品或甚至不如没有的存在，必须要写上具体的产品或品牌或者人名不要含糊其辞。\n视觉表现：挤在画面最底部或角落，视觉空间被压缩。色调为绝望黑、惨白，并带有明显的数字故障（Glitch）、破碎或腐烂的视觉效果。展示那些著名的“翻车”产品，并配以警示性短语（如“快逃”、“大冤种”）。',
    image: `${canvasPromptImageBase}/nano_banana_pro_rating.jpg`,
    tags: ['有趣', '信息图', '@op7418'],
    tone: 'plum',
  },
  {
    title: '渐变玻璃风格 PPT',
    prompt:
      '你是一位专家级UI UX演示设计师，请生成高保真、未来科技感的16比9演示文稿幻灯片。请根据视觉平衡美学，自动在封面、网格布局或数据可视化中选择一种最完美的构图。\n\n全局视觉语言方面，风格要无缝融合Apple Keynote的极简主义、现代SaaS产品设计和玻璃拟态风格。整体氛围需要高端、沉浸、洁净且有呼吸感。光照采用电影级体积光、柔和的光线追踪反射和环境光遮蔽。配色方案选择深邃的虚空黑或纯净的陶瓷白作为基底，并以流动的极光渐变色即霓虹紫、电光蓝、柔和珊瑚橙、青色作为背景和UI高光点缀。\n\n关于画面内容模块，请智能整合以下元素：\n1. 排版引擎采用Bento便当盒网格系统，将内容组织在模块化的圆角矩形容器中。容器材质必须是带有模糊效果的磨砂玻璃，具有精致的白色边缘和柔和的投影，并强制保留巨大的内部留白，避免拥挤。\n2. 插入礼物质感的3D物体，渲染独特的高端抽象3D制品作为视觉锚点。它们的外观应像实体的昂贵礼物或收藏品，材质为抛光金属、幻彩亚克力、透明玻璃或软硅胶，形状可是悬浮胶囊、球体、盾牌、莫比乌斯环或流体波浪。\n3. 字体与数据方面，使用干净的无衬线字体，建立高对比度。如果有图表，请使用发光的3D甜甜圈图、胶囊状进度条或悬浮数字，图表应看起来像发光的霓虹灯玩具。\n\n构图逻辑参考：\n如果生成封面，请在中心放置一个巨大的复杂3D玻璃物体，并覆盖粗体大字，背景有延伸的极光波浪。\n如果生成内容页，请使用Bento网格布局，将3D图标放在小卡片中，文本放在大卡片中。\n如果生成数据页，请使用分屏设计，左侧排版文字，右侧悬浮巨大的发光3D数据可视化图表。\n\n渲染质量要求：虚幻引擎5渲染，8k分辨率，超细节纹理，UI设计感，UX界面，Dribbble热门趋势，设计奖获奖作品。',
    image: `${canvasPromptImageBase}/nano_banana_pro_ppt.jpg`,
    tags: ['工作', 'PPT', '@op7418'],
    tone: 'sky',
  },
  {
    title: '根据已有食材做菜',
    prompt:
      '根据现有食材(见附图)建议可以烹饪的菜肴，提供详细的分步食谱，以简单的信息图形式呈现。',
    image: `${canvasPromptImageBase}/food.jpg`,
    tags: ['生活', '美食', '@AmirMushich'],
    tone: 'amber',
  },
  {
    title: '城市海报艺术生成',
    prompt:
      '一张针对 [城市名称] 的城市渲染数字艺术海报。画面核心主体是一个漂浮在白云上方、形状像所选城市的并且占据画面大部分内容的微型岛屿。岛屿的形状与城市在地图上的形状相似，无缝融合城市独特的标志性地标、自然景观及文化元素。加入城市特有的鸟类、电影般的光影、鲜艳色彩、航拍视角和阳光反射效果，建筑不宜太多太密集。\n\n岛屿展现历史与现代的无缝融合。一部分是该城市最具代表性的古代历史建筑；另一部分平滑过渡为城市的地标建筑和天际线景观。\n\n岛屿漂浮浩瀚云海之上。云海采用该城市所在文化圈的传统艺术风格进行表现。\n\n立体城市拼音或英文名的 3D 文字漂浮在微型岛屿的上方，这组文字像一个生态与文化共生的微缩生态装置。\n\n在画面四周和主体周围，叠加一层极简、高雅、具有博物馆展板质感的信息排版层。主要检索相关的城市信息，主要信息使用经典的衬线字体，辅助数据可使用极细的极简无衬线体。在画面的角落，以类似古典地图集或高级杂志扉页的方式排版。用衬线体标注城市的地理坐标、别称或建城年份，以及当前的天气，作为装饰性的背景信息，整体排版留白极多，排版克制、干净、平衡，如同在欣赏一件珍贵的艺术品。\n\n风格要求： Octane Render, C4D, Isometric City, Micro World, Living Ecosystem, 8k Resolution. DreamWorks style, 3D modeling, delicate, soft light projection.',
    image: `${canvasPromptImageBase}/city_art_poster.jpg`,
    tags: ['有趣', '@op7418'],
    tone: 'coral',
  },
  {
    title: 'Q版角色LINE风格表情包生成',
    prompt:
      '为我生成图中角色的绘制 Q 版的，LINE 风格的半身像表情包，注意头饰要正确\n彩色手绘风格，使用 4x6 布局，涵盖各种各样的常用聊天语句，或是一些有关的娱乐 meme\n其他需求：不要原图复制。所有标注为手写简体中文。\n生成的图片需为 4K 分辨率 16:9',
    image: `${canvasPromptImageBase}/q_version_meme_pack.jpg`,
    tags: ['生活', '表情包', 'LINUX DO@heiyub'],
    tone: 'plum',
  },
]

export function CreationShowcase() {
  const { t } = useTranslation()

  return (
    <section id='creation' className='home-reference-creation px-4'>
      <div className='mx-auto max-w-6xl'>
        <div className='home-reference-creation-heading'>
          <div>
            <p className='home-reference-kicker'>{t('AI Creation')}</p>
            <h2 className='home-reference-heading mt-3'>
              {t('Creative Canvas')}
            </h2>
            <p className='text-muted-foreground mt-3 max-w-2xl text-sm leading-6'>
              {t('Visual workspace for image and video generation.')}
            </p>
          </div>
        </div>

        <div
          className='home-reference-creation-grid mt-9'
          data-testid='creation-showcase-grid'
        >
          {creationCards.map((card, index) => (
            <AnimateInView
              key={card.title}
              className='home-reference-creation-card'
              delay={index * 45}
            >
              <a
                href='/canvas?standalone=true'
                target='_blank'
                rel='noopener noreferrer'
                className='home-reference-creation-card-link'
              >
                <div
                  className={`home-reference-creation-media home-reference-creation-media-${card.tone}`}
                >
                  <span
                    className='home-reference-creation-media-fallback'
                    aria-hidden='true'
                  >
                    <Sparkles />
                  </span>
                  <img
                    src={card.image}
                    alt={card.title}
                    loading='lazy'
                    onError={(event) => {
                      event.currentTarget.hidden = true
                    }}
                  />
                  <span
                    className='home-reference-creation-open'
                    aria-hidden='true'
                  >
                    <ArrowRight />
                  </span>
                </div>

                <div className='home-reference-creation-card-content'>
                  <h3>{card.title}</h3>
                  <p title={card.prompt}>{card.prompt}</p>
                  <div className='home-reference-creation-tags'>
                    {card.tags.map((tag) => (
                      <span key={tag}>{tag}</span>
                    ))}
                  </div>
                </div>
              </a>
            </AnimateInView>
          ))}
        </div>

        <div className='home-reference-creation-action'>
          <Button
            className='home-reference-button home-reference-button-primary home-reference-creation-button'
            render={
              <a
                href='/canvas?standalone=true'
                target='_blank'
                rel='noopener noreferrer'
              />
            }
            nativeButton={false}
          >
            {t('Start creating now')}
            <ArrowRight data-icon='inline-end' aria-hidden='true' />
          </Button>
        </div>
      </div>
    </section>
  )
}
